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
	Data   string `json:"data"`
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

	module, err := runtime.InstantiateWithConfig(context.Background(), code, moduleConfig)
	if err != nil {
		fmt.Printf("Error instantiating Wasm module: %v\n", err)
		return
	}

	defer module.Close(ctx) // Clean up resources when done.

	memory := module.Memory()
	if memory == nil {
		log.Println("Wasm module does not export memory")
		return
	}

	malloc := module.ExportedFunction("malloc")
	free := module.ExportedFunction("free")
	processTx := module.ExportedFunction("process")

	for tx := range inputStream {

		// Use malloc to allocate memory for the transaction data.
		namePtrResults, err := malloc.Call(ctx, uint64(len(tx)))
		if err != nil {
			log.Printf("malloc call failed: %v", err)
			continue
		}
		namePtr := namePtrResults[0]

		// Write the transaction data to the allocated space in memory.
		if !memory.Write(uint32(namePtr), tx) {
			log.Println("Failed to write transaction data to Wasm memory")
			continue
		}

		// Process the transaction.
		size, err := processTx.Call(ctx, namePtr, uint64(len(tx)))
		if err != nil {
			log.Printf("Process function call failed: %v", err)
			continue
		} else if size[0] == 0 {
			continue
		}

		resultData, ok := memory.Read(uint32(namePtr), uint32(size[0]))

		w.PublishWasmData(resultData, outputSubject)
		if !ok {
			log.Println("Failed to read from Wasm memory")
			continue
		}

		// Publish the processed data.
		err = w.Publisher.PublishTo(resultData, outputSubject)
		if err != nil {
			log.Printf("Failed to publish processed data: %v", err)
			continue
		}

		// Free the allocated memory.
		_, err = free.Call(ctx, namePtr)
		if err != nil {
			log.Printf("Failed to free allocated memory: %v", err)
		}
		fmt.Printf("Result of processed data: %s\n", resultData)
	}
}

func (w *Wasmlisher) PublishWasmData(data []byte, subject string) {

	var segments []Segment

	if err := json.Unmarshal(data, &segments); err != nil {
		// if not able to unmarshal into segments just try to publish everything
		_ = w.Publisher.PublishTo(data, subject)
		return
	}

	// Publish each segment to its corresponding subject
	for _, segment := range segments {
		err := w.Publisher.PublishTo([]byte(segment.Data), subject+"."+segment.Suffix)
		if err != nil {
			log.Printf("Failed to publish processed data for subject %s: %v", subject+"."+segment.Suffix, err)
		} else {
			fmt.Printf("Published data for subject %s\n", subject+"."+segment.Suffix)
		}
	}

}
