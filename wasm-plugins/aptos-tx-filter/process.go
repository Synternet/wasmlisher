package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unsafe"
)

type Segment struct {
	Suffix string `json:"suffix"`
	Data   string `json:"data"`
}

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
	outputData := ProcessAptosTransaction(inputData)

	copy(inputData, outputData)

	// Return the pointer to the allocated memory.
	return len(outputData)
}

func ProcessAptosTransaction(data []byte) []byte {
	var incoming AptosTransaction
	if err := json.Unmarshal(data, &incoming); err != nil {
		log.Printf("ERROR: %s", err.Error())
		return nil
	}

	result := []Segment{}

	for _, change := range incoming.Changes {
		eventType := extractEventType(change.Type)
		address := change.Address
		if address == "" {
			address = "empty"
		}
		suffix := fmt.Sprintf("%s.%s", address, eventType)
		changeData, err := json.Marshal(change)
		if err != nil {
			log.Printf("ERROR marshalling change: %s", err)
			continue
		}
		result = append(result, Segment{Suffix: suffix, Data: string(changeData)})
	}

	for _, event := range incoming.Events {
		eventType := extractEventType(event.Type)
		address := event.GUID.AccountAddress
		if address == "" {
			address = "empty"
		}
		suffix := fmt.Sprintf("%s.%s", address, eventType)
		eventData, err := json.Marshal(event)
		if err != nil {
			log.Printf("ERROR marshalling event: %s", err)
			continue
		}
		result = append(result, Segment{Suffix: suffix, Data: string(eventData)})
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		log.Printf("ERROR marshalling result: %s", err)
		return nil
	}

	return resultBytes
}

// extractEventType extracts the innermost type and removes generic type information if present
func extractEventType(eventType string) string {
	// Remove the namespace and extract the innermost type
	parts := strings.Split(eventType, "::")
	if len(parts) > 1 {
		eventType = parts[len(parts)-1]
	}

	// Remove generic type information if present
	index := strings.LastIndex(eventType, "<")
	if index != -1 {
		return eventType[:index]
	}

	// Remove the address if present
	index = strings.Index(eventType, ".")
	if index != -1 {
		return eventType[index+1:]
	}

	return eventType
}

func main() {}
