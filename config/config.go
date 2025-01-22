package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found, continuing with runtime environment variables.")
	}
	mandatoryEnvVars := []string{"BASE_URL", "UPDATES_BUCKET_NAME"}
	for _, envVar := range mandatoryEnvVars {
		value := GetEnv(envVar)
		if value == "" {
			log.Fatalf("Environment variable %s not set", envVar)
		}
	}
}

func GetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}
