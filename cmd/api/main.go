package main

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/router"
	"log"
	"net/http"
)

func init() {
	config.LoadConfig()
}

func main() {
	router := infrastructure.NewRouter()
	log.Println("Server is running on port 3000")
	err := http.ListenAndServe(":3000", router)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
