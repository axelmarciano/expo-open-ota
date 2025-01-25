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
	expoToken := GetEnv("EXPO_USERNAME")
	if expoToken == "" {
		log.Fatalf("EXPO_USERNAME not set")
	}
	if expoToken == "<EXPO_DEFAULT_USERNAME>" {
		log.Printf("EXPO_USERNAME is set to the default value, please replace it with a valid token")
	}
	jwtSecret := GetEnv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET not set")
	}
}

var DefaultEnvValues = map[string]string{
	"LOCAL_BUCKET_BASE_PATH": "./updates",
	"STORAGE_MODE":           "local",
	"BASE_URL":               "http://localhost:3000",
	"ENVIRONMENTS_LIST":      "staging,production",
	"PUBLIC_CERT_KEY_PATH":   "./certs/public-key.pem",
	"PRIVATE_CERT_KEY_PATH":  "./certs/private-key.pem",
	"CERTS_STORAGE_TYPE":     "local",
	"EXPO_USERNAME":          "<EXPO_DEFAULT_USERNAME>",
	"JWT_SECRET":             "",
}

func GetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		defaultValue := DefaultEnvValues[key]
		if defaultValue != "" {
			return defaultValue
		}
		return ""
	}
	return value
}
