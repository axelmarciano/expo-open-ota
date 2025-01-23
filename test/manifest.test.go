package test

import (
	"os"
	"testing"
)

func TestManifestHandler(t *testing.T) {
	// Set environment variable
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("BASE_URL", "http://localhost:3000")
}
