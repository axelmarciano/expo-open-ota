package config

import (
	"expo-open-ota/internal/modules/helpers"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
)

func validateStorageMode(storageMode string) bool {
	return storageMode == "local" || storageMode == "aws"
}

func GetEnvironmentsList() []string {
	environmentsList := GetEnv("ENVIRONMENTS_LIST")
	return strings.Split(environmentsList, ",")
}

func validateBaseUrl(baseUrl string) bool {
	return baseUrl != "" && helpers.IsValidURL(baseUrl)
}

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found, continuing with runtime environment variables.")
	}
	mandatoryEnvVars := []string{"BASE_URL", "S3_BUCKET_NAME"}
	for _, envVar := range mandatoryEnvVars {
		value := GetEnv(envVar)
		if value == "" {
			log.Fatalf("Environment variable %s not set", envVar)
		}
	}
	storageMode := GetEnv("STORAGE_MODE")
	if !validateStorageMode(storageMode) {
		log.Fatalf("Invalid STORAGE_MODE: %s", storageMode)
	}
	environmentsList := GetEnvironmentsList()
	if (len(environmentsList)) == 0 {
		log.Fatalf("No environments configured")
	}
	baseUrl := GetEnv("BASE_URL")
	if !validateBaseUrl(baseUrl) {
		log.Fatalf("Invalid BASE_URL: %s", baseUrl)
	}
}

var DefaultEnvValues = map[string]string{
	"LOCAL_BUCKET_BASE_PATH": "./updates",
	"STORAGE_MODE":           "local",
	"BASE_URL":               "http://localhost:3000",
	"ENVIRONMENTS_LIST":      "staging,production,demo",
}

func GetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		defaultValue := DefaultEnvValues[key]
		if defaultValue != "" {
			return defaultValue
		}
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}
