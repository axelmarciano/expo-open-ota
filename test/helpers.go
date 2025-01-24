package test

import (
	"encoding/json"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/types"
	"os"
	"path/filepath"
)

func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return "", os.ErrNotExist
}

func ComputeUploadRequestsInput(dirPath string) handlers.FileNamesRequest {
	// Retrieve metadata.json file
	metadataFilePath := filepath.Join(dirPath, "metadata.json")
	metadataFile, err := os.Open(metadataFilePath)
	if err != nil {
		panic(err)
	}
	defer metadataFile.Close()
	// Cast metadataFile as types.MetadataObject
	var metadataObject types.MetadataObject
	err = json.NewDecoder(metadataFile).Decode(&metadataObject)
	if err != nil {
		panic(err)
	}
	// Retrieve all file names from metadataObject
	fileNames := make([]string, 0)
	for _, asset := range metadataObject.FileMetadata.IOS.Assets {
		fileNames = append(fileNames, asset.Path)
	}
	for _, asset := range metadataObject.FileMetadata.Android.Assets {
		fileNames = append(fileNames, asset.Path)
	}
	fileNames = append(fileNames, metadataObject.FileMetadata.Android.Bundle)
	fileNames = append(fileNames, metadataObject.FileMetadata.IOS.Bundle)
	return handlers.FileNamesRequest{FileNames: fileNames}
}
