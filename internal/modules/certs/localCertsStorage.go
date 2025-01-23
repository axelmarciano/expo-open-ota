package certs

import (
	"fmt"
	"io"
	"os"
)

type LocalCertsStorage struct {
	privateKeyPath string
	publicKeyPath  string
}

func retrieveFileContent(path string) string {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return ""
	}
	defer file.Close()

	// Read the file content fully
	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return ""
	}

	return string(content)
}
func (c *LocalCertsStorage) GetPublicExpoCert() string {
	if c.publicKeyPath == "" {
		return ""
	}
	return retrieveFileContent(c.publicKeyPath)
}

func (c *LocalCertsStorage) GetPrivateExpoCert() string {
	if c.privateKeyPath == "" {
		return ""
	}
	private := retrieveFileContent(c.privateKeyPath)
	return private
}
