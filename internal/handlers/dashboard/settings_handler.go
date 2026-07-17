package handlers

import (
	"encoding/json"
	"expo-open-ota/config"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"net/http"
)

type SettingsHandler struct {
	appService *services.AppService
}

func NewSettingsHandler(appService *services.AppService) *SettingsHandler {
	return &SettingsHandler{
		appService: appService,
	}
}

type SettingsEnv struct {
	BASE_URL                               string `json:"BASE_URL"`
	CONTROL_PLANE_ENABLED                  bool   `json:"CONTROL_PLANE_ENABLED"`
	CACHE_MODE                             string `json:"CACHE_MODE"`
	REDIS_HOST                             string `json:"REDIS_HOST"`
	REDIS_PORT                             string `json:"REDIS_PORT"`
	STORAGE_MODE                           string `json:"STORAGE_MODE"`
	S3_BUCKET_NAME                         string `json:"S3_BUCKET_NAME"`
	LOCAL_BUCKET_BASE_PATH                 string `json:"LOCAL_BUCKET_BASE_PATH"`
	AWS_REGION                             string `json:"AWS_REGION"`
	AWS_BASE_ENDPOINT                      string `json:"AWS_BASE_ENDPOINT"`
	AWS_ACCESS_KEY_ID                      string `json:"AWS_ACCESS_KEY_ID"`
	CLOUDFRONT_DOMAIN                      string `json:"CLOUDFRONT_DOMAIN"`
	CLOUDFRONT_KEY_PAIR_ID                 string `json:"CLOUDFRONT_KEY_PAIR_ID"`
	PRIVATE_CLOUDFRONT_KEY_B64             string `json:"PRIVATE_CLOUDFRONT_KEY_B64"`
	AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID string `json:"AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"`
	PRIVATE_CLOUDFRONT_KEY_PATH            string `json:"PRIVATE_CLOUDFRONT_KEY_PATH"`
	PROMETHEUS_ENABLED                     string `json:"PROMETHEUS_ENABLED"`
	// Apps lists the configured apps — the single flat-env app in stateless
	// mode, or every app in the database in control-plane mode. Each entry
	// carries just the id and optional display name — tokens and keys are
	// never surfaced here because this endpoint is read by the dashboard UI.
	Apps []config.AppDescriptor `json:"APPS"`
}

func (h *SettingsHandler) GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	apps, err := h.appService.GetApps(r.Context())
	if err != nil {
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while fetching app settings")
		return
	}
	marshaledResponse, _ := json.Marshal(SettingsEnv{
		BASE_URL:                               config.GetEnv("BASE_URL"),
		CONTROL_PLANE_ENABLED:                  config.IsDBMode(),
		CACHE_MODE:                             config.GetEnv("CACHE_MODE"),
		REDIS_HOST:                             config.GetEnv("REDIS_HOST"),
		REDIS_PORT:                             config.GetEnv("REDIS_PORT"),
		STORAGE_MODE:                           config.GetEnv("STORAGE_MODE"),
		S3_BUCKET_NAME:                         config.GetEnv("S3_BUCKET_NAME"),
		LOCAL_BUCKET_BASE_PATH:                 config.GetEnv("LOCAL_BUCKET_BASE_PATH"),
		AWS_REGION:                             config.GetEnv("AWS_REGION"),
		AWS_BASE_ENDPOINT:                      config.GetEnv("AWS_BASE_ENDPOINT"),
		AWS_ACCESS_KEY_ID:                      helpers.MaskSecret(config.GetEnv("AWS_ACCESS_KEY_ID")),
		CLOUDFRONT_DOMAIN:                      config.GetEnv("CLOUDFRONT_DOMAIN"),
		CLOUDFRONT_KEY_PAIR_ID:                 helpers.MaskSecret(config.GetEnv("CLOUDFRONT_KEY_PAIR_ID")),
		PRIVATE_CLOUDFRONT_KEY_B64:             helpers.MaskSecret(config.GetEnv("PRIVATE_CLOUDFRONT_KEY_B64")),
		AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID: config.GetEnv("AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"),
		PRIVATE_CLOUDFRONT_KEY_PATH:            config.GetEnv("PRIVATE_CLOUDFRONT_KEY_PATH"),
		PROMETHEUS_ENABLED:                     config.GetEnv("PROMETHEUS_ENABLED"),
		Apps:                                   apps,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaledResponse)

}
