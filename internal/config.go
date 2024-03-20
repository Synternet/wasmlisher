package wasmlisher

import (
	"encoding/json"
	"fmt"
	"os"
)

// StreamConf represents configuration of our secondary streams.
type StreamConf struct {
	InputStream  string `json:"input"`
	OutputStream string `json:"output"`
	File         string `json:"file"`
	Type         string `json:"type"`
}

// LoadConfig reads and parses the configuration from a given file path.
// It returns a slice of Config objects and an error, if any occurred during file reading or parsing.
func LoadConfig(filePath string) ([]StreamConf, error) {

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse JSON content
	var streams []StreamConf
	err = json.Unmarshal(fileContent, &streams)
	if err != nil {
		return nil, fmt.Errorf("error parsing config JSON: %w", err)
	}

	return streams, nil
}
