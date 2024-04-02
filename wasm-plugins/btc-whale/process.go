package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"unsafe"
)

// Helper function to allocate memory in the Wasm environment
//
//go:linkname malloc runtime.malloc
func malloc(size uintptr) unsafe.Pointer

// process is the exported function that can be called from the host environment.
// It expects a pointer to the input data and the size of the data.
// It processes the data and returns a pointer to the result data allocated in the WebAssembly memory.
//
//export process
func process(ptr *byte, size int) int {
	// Convert the input pointer and size to a Go byte slice.
	inputData := unsafe.Slice(ptr, size)

	// Process the input data and get the output data as a byte slice.
	outputData := ProcessBitcoinTransaction(inputData)

	copy(inputData[:len(outputData)], outputData)

	// Return the pointer to the allocated memory.
	return len(outputData)
}

func ProcessBitcoinTransaction(data []byte) []byte {
	var incoming BitcoinTransaction
	if err := json.Unmarshal(data, &incoming); err != nil {
		log.Printf("ERROR: %s", err.Error())
		return nil
	}

	filteredMessages := []FilteredMessage{}

	thresholdStr := os.Getenv("THRESHOLD")
	threshold, _ := strconv.ParseFloat(thresholdStr, 64)

	for _, vout := range incoming.Vout {
		if vout.Value >= threshold {
			filteredMessage := FilteredMessage{
				TxID:  incoming.Txid,
				Value: vout.Value,
				To:    vout.ScriptPubKey.Address,
			}
			filteredMessages = append(filteredMessages, filteredMessage)
		}
	}

	if len(filteredMessages) == 0 {
		return nil // No transactions met the filter criteria
	}

	// Convert the filtered messages back to JSON to return.
	jsonBytes, err := json.Marshal(filteredMessages)
	if err != nil {
		log.Printf("ERROR: Could not marshal filtered messages: %s", err)
		return nil
	}

	return jsonBytes
}

func main() {}
