package handlers

import (
	"encoding/json"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"net/http"

	"github.com/gorilla/mux"
)

type UpdateHandler struct {
	updateService *services.UpdateService
}

func NewUpdateHandler(updateService *services.UpdateService) *UpdateHandler {
	return &UpdateHandler{
		updateService: updateService,
	}
}

func (h *UpdateHandler) GetUpdateDetailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["APP_ID"]
	branchName := vars["BRANCH"]
	runtimeVersion := vars["RUNTIME_VERSION"]
	updateId := vars["UPDATE_ID"]
	if branchName == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Branch name is empty")
		return
	}
	if runtimeVersion == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Runtime version is empty")
		return
	}
	if updateId == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Update ID is empty")
		return
	}
	cacheKey := dashboard.ComputeGetUpdateDetailsCacheKey(appId, branchName, runtimeVersion, updateId)
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cacheValue))
		return
	}
	update, err := h.updateService.GetUpdateDetails(r.Context(), appId, branchName, runtimeVersion, updateId)
	if err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "An internal error occurred while fetching update details.")
		return
	}
	updatesResponse := types.UpdateDetails{
		UpdateUUID: update.UpdateUUID,
		UpdateId:   update.UpdateId,
		CreatedAt:  update.CreatedAt,
		CommitHash: update.CommitHash,
		Platform:   update.Platform,
		Message:    update.Message,
		Type:       update.Type,
		ExpoConfig: update.ExpoConfig,
	}
	marshaledResponse, _ := json.Marshal(updatesResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaledResponse)

	ttl := 604800 // 7 days
	cache.Set(cacheKey, string(marshaledResponse), &ttl)
}

func (h *UpdateHandler) GetUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["APP_ID"]
	branchName := vars["BRANCH"]
	runtimeVersion := vars["RUNTIME_VERSION"]
	if branchName == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Branch name is empty")
		return
	}
	if runtimeVersion == "" {
		handlers.RenderError(w, http.StatusBadRequest, "Runtime version is empty")
		return
	}
	cacheKey := dashboard.ComputeGetUpdatesCacheKey(appId, branchName, runtimeVersion)
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cacheValue))
		return
	}
	updates, err := h.updateService.GetUpdatesByRunTimeVersionAndBranchName(r.Context(), appId, runtimeVersion, branchName)
	if err != nil {
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while fetching updates.")
		return
	}
	marshaledResponse, _ := json.Marshal(updates)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaledResponse)

	ttl := 3600
	cache.Set(cacheKey, string(marshaledResponse), &ttl)
}
