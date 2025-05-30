package main

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/migration"
	infrastructure "expo-open-ota/internal/router"
	"github.com/gorilla/handlers"
	"log"
	"net/http"
)

import (
	_ "expo-open-ota/internal/migrations"
)

func init() {
	config.LoadConfig()
	metrics.InitMetrics()
}

func main() {
	migration.RunMigrationsWithLock()
	router := infrastructure.NewRouter()
	log.Println("Server is running on port " + config.GetPort())
	corsOptions := handlers.CORS(
		handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	err := http.ListenAndServe("0.0.0.0:"+config.GetPort(), corsOptions(router))
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
