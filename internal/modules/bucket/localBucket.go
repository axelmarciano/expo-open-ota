package bucket

import (
	"errors"
	"expo-open-ota/config"
	"expo-open-ota/internal/modules/types"
	"expo-open-ota/internal/services"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type LocalBucket struct {
	BasePath string
}

func (b *LocalBucket) RequestUploadUrlForFileUpdate(environment string, runtimeVersion string, updateId string, fileName string) (string, error) {
	if b.BasePath == "" {
		return "", errors.New("BasePath not set")
	}
	fmt.Println("BasePath", b.BasePath)
	fmt.Println("environment", environment)
	fmt.Println("runtimeVersion", runtimeVersion)
	fmt.Println("updateId", updateId)
	dirPath := filepath.Join(b.BasePath, environment, runtimeVersion, updateId)
	fmt.Println("dirPath", dirPath)
	fmt.Println("fileName", fileName)
	fmt.Println("composed", filepath.Join(dirPath, fileName))
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	token, err := services.GenerateJWTToken(config.GetEnv("JWT_SECRET"), jwt.MapClaims{
		"sub":      config.GetEnv("EXPO_USERNAME"),
		"exp":      time.Now().Add(time.Minute * 10).Unix(),
		"filePath": filepath.Join(dirPath, fileName),
		"action":   "uploadLocalFile",
	})
	if err != nil {
		return "", err
	}
	parsedURL, err := url.Parse(config.GetEnv("BASE_URL"))
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	parsedURL.Path, err = url.JoinPath(parsedURL.Path, "uploadLocalFile")
	if err != nil {
		return "", fmt.Errorf("error joining path: %w", err)
	}
	query := url.Values{}
	query.Set("token", token)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func (b *LocalBucket) GetUpdates(environment string, runtimeVersion string) ([]types.Update, error) {
	if b.BasePath == "" {
		return nil, errors.New("BasePath not set")
	}
	dirPath := filepath.Join(b.BasePath, environment, runtimeVersion)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return []types.Update{}, nil
	}
	var updates []types.Update
	for _, entry := range entries {
		if entry.IsDir() {
			updateId, err := strconv.ParseInt(entry.Name(), 10, 64)
			if err == nil {
				updates = append(updates, types.Update{
					Environment:    environment,
					RuntimeVersion: runtimeVersion,
					UpdateId:       strconv.FormatInt(updateId, 10),
					CreatedAt:      time.Duration(updateId) * time.Millisecond,
				})
			}
		}
	}
	return updates, nil
}

func (b *LocalBucket) GetFile(update types.Update, assetPath string) (types.BucketFile, error) {
	if b.BasePath == "" {
		return types.BucketFile{}, errors.New("BasePath not set")
	}

	filePath := filepath.Join(b.BasePath, update.Environment, update.RuntimeVersion, update.UpdateId, assetPath)

	file, err := os.Open(filePath)
	if err != nil {
		return types.BucketFile{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return types.BucketFile{}, err
	}
	return types.BucketFile{
		Reader:    file,
		CreatedAt: fileInfo.ModTime(),
	}, nil
}

func ValidateUploadTokenAndResolveFilePath(token string) (string, error) {
	claims := jwt.MapClaims{}
	decodedToken, err := services.DecodeAndExtractJWTToken(config.GetEnv("JWT_SECRET"), token, claims)
	if err != nil {
		return "", err
	}
	if !decodedToken.Valid {
		return "", errors.New("invalid token")
	}
	action := claims["action"].(string)
	filePath := claims["filePath"].(string)
	sub := claims["sub"].(string)
	if sub != config.GetEnv("EXPO_USERNAME") {
		return "", errors.New("invalid token sub")
	}
	if action != "uploadLocalFile" {
		return "", errors.New("invalid token action")
	}
	return filePath, nil
}

func HandleUploadFile(filePath string, body multipart.File) (bool, error) {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return false, err
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println(filePath)
		return false, err
	}
	defer file.Close()
	_, err = io.Copy(file, body)
	if err != nil {
		return false, err
	}
	return true, nil
}
