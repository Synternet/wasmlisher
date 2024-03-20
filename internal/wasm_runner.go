package wasmlisher

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"os"
)

// runWasmWithInput instantiates the module and processes each input from the channel.
func (w *Wasmlisher) RunWasmStream(wasmFilePath string, inputStream <-chan []byte, outputSubject string) {

	ctx := context.Background()

	// Initialize wazero runtime
	runtime := wazero.NewRuntime(ctx)

	// Load WebAssembly module
	code, err := os.ReadFile(wasmFilePath)
	if err != nil {
		panic(err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	module, err := runtime.Instantiate(context.Background(), code)
	if err != nil {
		panic(err)
	}

	memory := module.Memory()
	if memory == nil {
		panic("Wasm module does not export memory")
	}

	// Iterate over the stream of transactions
	for tx := range inputStream {
		offset := uint32(0)
		ok := memory.Write(offset, tx)
		if !ok {
			panic("failed to write to Wasm memory")
		}

		// Retrieve the exported function "process"
		processTx := module.ExportedFunction("process")

		_, err := processTx.Call(context.Background(), uint64(offset), uint64(len(tx)))
		if err != nil {
			panic(err)
		}

		// Read the processed data from memory
		resultData, ok := memory.Read(offset, uint32(len(tx)))
		if !ok {
			panic("failed to read from Wasm memory")
		}

		// Process the result
		w.Publisher.PublishTo(resultData, outputSubject)
		fmt.Printf("Result of processed data for %s: %s\n", tx, resultData)
	}
}
