package wasmlisher

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"log"
	"os"
)

func (w *Wasmlisher) RunWasmStream(wasmFilePath string, inputStream <-chan []byte, outputSubject string) {
	ctx := context.Background()

	// Initialize wazero runtime
	runtime := wazero.NewRuntime(ctx)

	// Load WebAssembly module
	code, err := os.ReadFile(wasmFilePath)
	if err != nil {
		log.Printf("Failed to read wasm file: %v", err)
		return
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	module, err := runtime.Instantiate(context.Background(), code)
	if err != nil {
		log.Printf("Failed to instantiate wasm module: %v", err)
		return
	}

	memory := module.Memory()
	if memory == nil {
		log.Println("Wasm module does not export memory")
		return
	}

	memory.Grow(100) // memory should be enough for any message. memory is overwritten each time.

	// Iterate over the stream of transactions
	for tx := range inputStream {
		fmt.Println("tx")
		offset := uint32(0)
		ok := memory.Write(offset, tx)
		if !ok {
			log.Println("Failed to write to Wasm memory")
			return
		}

		// Retrieve the exported function "process"
		processTx := module.ExportedFunction("process")

		_, err := processTx.Call(context.Background(), uint64(offset), uint64(len(tx)))
		if err != nil {
			log.Printf("Process function call failed: %v", err)
			return
		}

		// Read the processed data from memory
		resultData, ok := memory.Read(offset, uint32(len(tx)))
		if !ok {
			log.Println("Failed to read from Wasm memory")
			return
		}

		// Process the result
		err = w.Publisher.PublishTo(resultData, outputSubject)
		if err != nil {
			log.Printf("Failed to publish processed data: %v", err)
			return
		}
		fmt.Printf("Result of processed data: %s\n", resultData)
	}
}
