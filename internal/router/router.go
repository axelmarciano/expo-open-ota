package infrastructure

import (
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/middleware"
	"github.com/gorilla/mux"
	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)
	r.HandleFunc("/hc", HealthCheck).Methods(http.MethodGet)
	r.HandleFunc("/manifest/{ENVIRONMENT}", handlers.ManifestHandler).Methods(http.MethodGet)
	r.HandleFunc("/assets/{ENVIRONMENT}", handlers.AssetsHandler).Methods(http.MethodGet)
	return r
}
