package wasmlisher

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// StreamConf represents configuration of our secondary streams.
type StreamConf struct {
	InputStream  string            `json:"input"`
	OutputStream string            `json:"output"`
	File         string            `json:"file"`
	Type         string            `json:"type"`
	Env          map[string]string `json:"env"`
	LocalPath    string
}

// Determine if the config string is a URL or a path
func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func FindStreamByInput(streams []StreamConf, inputStream string) (StreamConf, bool) {
	for _, stream := range streams {
		if stream.InputStream == inputStream {
			return stream, true
		}
	}
	return StreamConf{}, false
}

func LoadConfig(config string, existingStreams []StreamConf) ([]StreamConf, error) {
	var streams []StreamConf
	var err error

	if isURL(config) {
		streams, err = LoadConfigFromUrl(config)
	} else {
		streams, err = LoadConfigFromFile(config)
	}

	if err != nil {
		return nil, err
	}

	for i, stream := range streams {
		if stream.Type == "ipfs" {
			existingStream, exists := FindStreamByInput(existingStreams, stream.InputStream)
			if !exists || existingStream.File != stream.File {
				localPath, err := DownloadFile(stream.File)
				if err != nil {
					return nil, fmt.Errorf("error downloading IPFS file: %w", err)
				}
				streams[i].LocalPath = localPath
			} else {
				// If the file has not changed, use the previously downloaded path
				streams[i].LocalPath = existingStream.LocalPath
			}
		} else {
			streams[i].LocalPath = stream.File
		}
	}

	return streams, nil
}

// LoadConfig reads and parses the configuration from a given file path.
// It returns a slice of Config objects and an error, if any occurred during file reading or parsing.
func LoadConfigFromFile(filePath string) ([]StreamConf, error) {

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

// LoadConfigFromUrl reads and parses the configuration from a given URL.
// It returns a slice of Config objects and an error, if any occurred during fetching or parsing.
func LoadConfigFromUrl(url string) ([]StreamConf, error) {
	// Fetch the content from the URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching config from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Parse JSON content
	var streams []StreamConf
	err = json.Unmarshal(body, &streams)
	if err != nil {
		return nil, fmt.Errorf("error parsing config JSON: %w", err)
	}

	return streams, nil
}

// DownloadFile downloads a file from the given URL and saves it to a local temporary file.
// Returns the path to the local file or an error.
func DownloadFile(fileURL string) (string, error) {
	// Get the data
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Create the file in the temp directory
	tmpDir := os.TempDir()
	fileName := filepath.Base(fileURL)
	tmpFile, err := os.CreateTemp(tmpDir, fileName+"-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write the body to file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
