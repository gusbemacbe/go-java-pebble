package fs

import (
	"os"
)

// The `ReadFile` function reads the content of a file at the given path
func ReadFile(path string) ([]byte, error) {
	// Using `os.ReadFile` which is the recommended way to read an entire file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}
