package wasmlisher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"log"
	"os"
)

type Segment struct {
	Suffix string `json:"suffix"`
	Data   any    `json:"data"`
}

func (w *Wasmlisher) RunWasmStream(wasmFilePath string, inputStream <-chan []byte, outputSubject string, env map[string]string) {

	ctx := context.Background()

	// Initialize wazero runtime
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx) // Clean up resources when done.

	// Load WebAssembly module
	code, err := os.ReadFile(wasmFilePath)
	if err != nil {
		log.Printf("Failed to read wasm file: %v", err)
		return
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	moduleConfig := wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr)
	for key, value := range env {
		moduleConfig = moduleConfig.WithEnv(key, value)
	}

	for tx := range inputStream {

		module, err := runtime.InstantiateWithConfig(context.Background(), code, moduleConfig)
		if err != nil {
			log.Printf("Failed to instantiate module: %v", err)
			return
		}

		malloc := module.ExportedFunction("malloc")

		// Use malloc to allocate memory for the transaction data.
		namePtrResults, err := malloc.Call(ctx, uint64(1000000)) // this should be enough for any transaction data.
		if err != nil {
			log.Printf("malloc call failed: %v", err)
		}
		namePtr := namePtrResults[0]

		memory := module.Memory()
		if memory == nil {
			log.Println("Wasm module does not export memory")
			return
		}
		// Write the transaction data to the allocated space in memory.
		if !memory.Write(uint32(namePtr), tx) {
			log.Println("Failed to write transaction data to Wasm memory")
			continue
		}
		processTx := module.ExportedFunction("process")
		// Process the transaction.
		size, err := processTx.Call(ctx, namePtr, uint64(len(tx)))

		if err != nil {
			log.Printf("Process function call failed: %v", err)
			continue
		} else if size[0] == 0 {
			continue
		}

		resultData, ok := memory.Read(uint32(namePtr), uint32(size[0]))
		if !ok {
			log.Println("Failed to read from Wasm memory")
			continue
		}
		if len(resultData) != 0 {
			w.PublishWasmData(resultData, outputSubject)
		}
		go module.Close(ctx)
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
			err := w.Publisher.PublishTo(segment.Data, segmentSubject)
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
