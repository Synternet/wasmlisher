package main

import (
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"
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
	outputData := ProcessAmountMessageFromStream(inputData)

	copy(inputData, outputData)

	// Return the pointer to the allocated memory.
	return len(outputData)
}

func ProcessAmountMessageFromStream(data []byte) []byte {
	var incoming OsmosisTransaction
	if err := json.Unmarshal(data, &incoming); err != nil {
		log.Printf("ERROR: %s", err.Error())
		return nil
	}

	messages := incoming.Tx.Body.Messages
	if len(messages) == 0 {
		return nil
	}

	messageType := messages[0].Type

	validMessageTypes := []string{
		"/osmosis.gamm.v1beta1.MsgSwapExactAmountIn",
		"/osmosis.gamm.v1beta1.MsgSwapExactAmountOut",
		"/osmosis.poolmanager.v1beta1.MsgSwapExactAmountIn",
		"/osmosis.poolmanager.v1beta1.MsgSwapExactAmountOut",
	}

	if !contains(validMessageTypes, messageType) {
		return nil
	}

	message := messages[0]
	routes := message.Routes
	routeLen := len(routes)

	tokenOutDenom, tokenInDenom := "", ""
	if routeLen > 0 && strings.Contains(messageType, "MsgSwapExactAmountIn") {
		if routes[routeLen-1].TokenOutDenom != nil {
			tokenOutDenom = *routes[routeLen-1].TokenOutDenom
		}
	} else if routeLen > 0 && strings.Contains(messageType, "MsgSwapExactAmountOut") {
		if routes[0].TokenInDenom != nil {
			tokenInDenom = *routes[0].TokenInDenom
		}
	}

	logResult := incoming.TxResult.Result.Log
	tokenOutAmount, tokenInAmount := "", ""

	if tokenOutDenom != "" && strings.Contains(messageType, "MsgSwapExactAmountIn") {
		re := regexp.MustCompile(`"tokens_out","value":"([0-9]+)` + tokenOutDenom + `"`)
		match := re.FindStringSubmatch(logResult)
		if len(match) > 0 {
			tokenOutAmount = match[1]
		}
	} else if tokenInDenom != "" && strings.Contains(messageType, "MsgSwapExactAmountOut") {
		re := regexp.MustCompile(`"amount","value":"([0-9]+)` + tokenInDenom + `"`)
		match := re.FindStringSubmatch(logResult)
		if len(match) > 0 {
			tokenInAmount = match[1]
		}
	}

	if strings.Contains(messageType, "MsgSwapExactAmountIn") && message.TokenIn != nil {
		tokenInAmount = message.TokenIn.Amount
		tokenInDenom = message.TokenIn.Denom
	} else if strings.Contains(messageType, "MsgSwapExactAmountOut") && message.TokenOut != nil {
		tokenOutAmount = message.TokenOut.Amount
		tokenOutDenom = message.TokenOut.Denom
	}

	filteredMessage := FilteredMessage{
		TxHash:         incoming.TxID,
		Address:        message.Sender,
		Code:           incoming.Code,
		TokenOutDenom:  tokenOutDenom,
		TokenOutAmount: tokenOutAmount,
		TokenInAmount:  tokenInAmount,
		TokenInDenom:   tokenInDenom,
		Timestamp:      int(time.Now().Unix()),
	}

	tokenInDenomName, err := GetBaseDenomName(incoming, filteredMessage.TokenInDenom)
	if err == nil {
		filteredMessage.TokenInDenomName = tokenInDenomName
	}

	tokenOutDenomName, err := GetBaseDenomName(incoming, filteredMessage.TokenOutDenom)
	if err == nil {
		filteredMessage.TokenOutDenomName = tokenOutDenomName
	}

	jsonBytes, err := json.Marshal(filteredMessage)

	return jsonBytes
}

func GetBaseDenomName(t OsmosisTransaction, key string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(key), "ibc/") {
		return key, nil
	}
	if trace, ok := t.Metadata[key]; ok {
		return trace.BaseDenom, nil
	}
	return "", errors.New("base_denom not found")
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func main() {}
