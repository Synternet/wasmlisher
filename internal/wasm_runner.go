package wasmlisher

import (
	"encoding/json"
	"fmt"
	wasmtimego "github.com/bytecodealliance/wasmtime-go/v21"
	"io/ioutil"
	"log"
	"log/slog"
)

type Segment struct {
	Suffix string `json:"suffix"`
	Data   any    `json:"data"`
}

func (w *Wasmlisher) RunWasmStream(wasmFilePath string, inputStream <-chan []byte, outputSubject string, env map[string]string) {
	// Read the WebAssembly file
	code, err := ioutil.ReadFile(wasmFilePath)
	if err != nil {
		log.Fatalf("Failed to read wasm file: %v", err)
	}

	engine := wasmtimego.NewEngine()

	store := wasmtimego.NewStore(engine)

	module, err := wasmtimego.NewModule(engine, code)
	if err != nil {
		log.Fatalf("Failed to compile module: %v", err)
	}

	wasiConfig := wasmtimego.NewWasiConfig()

	store.SetWasi(wasiConfig)

	linker := wasmtimego.NewLinker(engine)
	err = linker.DefineWasi()
	if err != nil {
		log.Fatalf("Failed to define WASI: %v", err)
	}

	instance, err := linker.Instantiate(store, module)
	if err != nil {
		log.Fatalf("Failed to instantiate module: %v", err)
	}

	alloc := instance.GetExport(store, "malloc").Func()
	if alloc == nil {
		log.Fatalf("Failed to get malloc function")
	}

	process := instance.GetExport(store, "process").Func()
	if process == nil {
		log.Fatalf("Failed to get process function")
	}

	// Access the memory
	memory := instance.GetExport(store, "memory").Memory()
	if memory == nil {
		log.Fatalf("Failed to get memory")
	}

	memory.Grow(store, 50)
	memoryData := memory.UnsafeData(store)
	memorySize := int32(len(memoryData))
	fmt.Printf("Memory size: %d bytes\n", memorySize)

	const memoryBlockSize = 55000 // Adjust this based on your needs

	if memoryBlockSize > memorySize {
		log.Fatalf("Memory block size %d exceeds memory size %d", memoryBlockSize, memorySize)
	}

	// Allocate a fixed memory block once
	ptrVal, err := alloc.Call(store, memoryBlockSize)
	if err != nil {
		log.Fatalf("Failed to allocate memory: %v", err)
	}
	ptr := ptrVal.(int32)

	// Ensure ptr is within bounds
	if ptr < 0 || ptr+memoryBlockSize > memorySize {
		log.Fatalf("Allocated pointer is out of memory bounds: %d", ptr)
	}
	// Process each transaction from the input stream
	for tx := range inputStream {
		txSize := int32(len(tx))
		if txSize > memoryBlockSize {
			log.Printf("Transaction size %d exceeds allocated memory block size %d", txSize, memoryBlockSize)
			continue
		}
		store.GC()
		// Zero out the allocated memory block before copying new data
		for i := int32(0); i < memoryBlockSize; i++ {
			memoryData[ptr+i] = 0
		}

		// Copy the transaction data to the allocated space in memory
		copy(memoryData[ptr:ptr+txSize], tx)

		// Process the transaction
		resultVal, err := process.Call(store, ptr, txSize)
		if err != nil {
			log.Printf("Process function call failed: %v", err)
			continue
		}
		size := resultVal.(int32)
		if size == 0 {
			continue
		}

		resultData := memoryData[ptr : ptr+size]

		w.PublishWasmData(resultData, outputSubject)
	}
}

func (w *Wasmlisher) PublishWasmData(data []byte, subject string) {
	// Try to unmarshal the data into the expected segments structure.
	var segments []Segment
	err := json.Unmarshal(data, &segments)
	// If unmarshaling into segments is successful, publish each segment.
	if err == nil {
		// Data unmarshaled successfully, publish each segment.
		for _, segment := range segments {
			segmentSubject := subject + "." + segment.Suffix
			msgBytes, err := json.Marshal(segment.Data)
			if err != nil {
				slog.Error("Failed to serialize message:", err)
				continue
			}

			err = w.Publisher.PublishBufTo(msgBytes, segmentSubject)
			if err != nil {
				log.Printf("Failed to publish processed data for subject %s: %v", segmentSubject, err)
			} else {
				fmt.Printf("Published segmented data for subject %s\n", segmentSubject)
			}
		}
	} else {
		// If no segmentation, publish the data as is.
		test := string(data)
		err := w.Publisher.PublishBufTo([]byte(test), subject)
		if err != nil {
			log.Printf("Failed to publish processed data for subject %s: %v", subject, err)
		} else {
			fmt.Printf("Published full data for subject %s\n", subject)
		}
	}
}
