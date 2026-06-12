package handlers

import (
	"encoding/json"
	"errors"
	"expo-open-ota/config"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/store"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type AppHandler struct {
	appService *services.AppService
}

func NewAppHandler(appService *services.AppService) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

func (h *AppHandler) CreateAppHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Name       string            `json:"name"`
		KeysConfig config.KeysConfig `json:"keysConfig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if requestBody.Name == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Display name is empty")
		return
	}
	appId, err := h.appService.CreateApp(r.Context(), requestBody.Name, requestBody.KeysConfig)
	if err != nil {
		if alreadyExistsErr := (*store.ErrResourceAlreadyExists)(nil); errors.As(err, &alreadyExistsErr) {
			handlers.RenderError(w, http.StatusConflict, alreadyExistsErr.Error())
			return
		}
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while creating the app.")
		return
	}

	marshaledResponse, _ := json.Marshal(map[string]interface{}{
		"appId": appId,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(marshaledResponse)

	cache := cache2.GetCache()
	cacheKey := dashboard.ComputeGetAppsCacheKey()
	cache.Delete(cacheKey)
}

func (h *AppHandler) DeleteAppHandler(w http.ResponseWriter, r *http.Request) {
	appId := mux.Vars(r)["APP_ID"]
	if err := config.ValidateAppId(appId, "APP_ID"); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, err.Error())
		return
	}
	err := h.appService.DeleteApp(r.Context(), appId)
	if err != nil {
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			handlers.RenderError(w, http.StatusNotFound, notFoundErr.Error())
			return
		}
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while deleting the app.")
		return
	}
	w.WriteHeader(http.StatusNoContent)

	cache := cache2.GetCache()
	cacheKey := dashboard.ComputeGetAppsCacheKey()
	cache.Delete(cacheKey)
}

func (h *AppHandler) GetAppsHandler(w http.ResponseWriter, r *http.Request) {
	cache := cache2.GetCache()
	cacheKey := dashboard.ComputeGetAppsCacheKey()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cacheValue))
		return
	}
	apps, err := h.appService.GetApps(r.Context())
	if err != nil {
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while fetching apps.")
		return
	}

	marshaledResponse, _ := json.Marshal(apps)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaledResponse)

	ttl := 10 * time.Second
	ttlMs := int(ttl.Milliseconds())
	cache.Set(cacheKey, string(marshaledResponse), &ttlMs)
}

func (h *AppHandler) UpdateAppHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["APP_ID"]

	if err := config.ValidateAppId(appId, "APP_ID"); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, err.Error())
		return
	}

	var requestBody struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if requestBody.Name == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Display name cannot be empty")
		return
	}

	err := h.appService.UpdateApp(r.Context(), appId, requestBody.Name)
	if err != nil {
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			handlers.RenderError(w, http.StatusNotFound, notFoundErr.Error())
			return
		}
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while updating the app.")
		return
	}

	cache := cache2.GetCache()
	appsCacheKey := dashboard.ComputeGetAppsCacheKey()
	cache.Delete(appsCacheKey)

	w.WriteHeader(http.StatusNoContent)
}

func (h *AppHandler) DownloadAppCertificateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["APP_ID"]

	if err := config.ValidateAppId(appId, "APP_ID"); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, err.Error())
		return
	}

	pemCertificateString, err := h.appService.RetrieveAppCertificate(r.Context(), appId)
	if err != nil {
		if notFoundErr := (*store.ErrResourceNotFound)(nil); errors.As(err, &notFoundErr) {
			handlers.RenderError(w, http.StatusNotFound, notFoundErr.Error())
			return
		}
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while downloading the app certificate.")
		return
	}

	cache := cache2.GetCache()
	appsCacheKey := dashboard.ComputeGetAppsCacheKey()
	cache.Delete(appsCacheKey)

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", `attachment; filename="certificate.pem"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pemCertificateString)))
	w.Header().Set("Cache-Control", "private, no-cache, no-store")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(pemCertificateString))
}
