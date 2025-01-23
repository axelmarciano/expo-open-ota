package certs

import "os"

type LocalCertsStorage struct {
	privateKeyPath string
	publicKeyPath  string
}

func retrieveFileContent(path string) string {
	// Open the file and read the content
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil {
		return ""
	}
	return string(buf[:n])
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
	return retrieveFileContent(c.privateKeyPath)
}
