// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package apikeyscopes

import (
	"encoding/json"
	"errors"
	"expo-open-ota/internal/cache"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/version"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Same shape as the internal/dashboard cache keys (appId included so entries
// from one app are never served to another); it lives here because the
// feature is enterprise code.
func computeGetApiKeyScopesCacheKey(appId string) string {
	return fmt.Sprintf("dashboard:%s:%s:request:getApiKeyScopes", version.Version, appId)
}

type ApiKeyScopeHandler struct {
	service *ApiKeyScopeService
}

func NewApiKeyScopeHandler(service *ApiKeyScopeService) *ApiKeyScopeHandler {
	return &ApiKeyScopeHandler{service: service}
}

// ApiKeyScopesResponse mirrors the dashboard's id conventions: every id is a
// string, like ApiKeyMetadata.ID and ChannelMapping.ReleaseChannelId.
type ApiKeyScopesResponse struct {
	ApiKeyID   string   `json:"apiKeyId"`
	ChannelIDs []string `json:"channelIds"`
	AllowedIps []string `json:"allowedIps"`
}

func renderApiKeyScopeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRequiresControlPlane), errors.Is(err, ErrInvalidCidr):
		handlers.RenderError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrRequiresValidLicense):
		handlers.RenderError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrApiKeyNotFound), errors.Is(err, ErrChannelNotFound):
		handlers.RenderError(w, http.StatusNotFound, err.Error())
	default:
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred.")
	}
}

func (h *ApiKeyScopeHandler) GetApiKeyScopesHandler(w http.ResponseWriter, r *http.Request) {
	appId := mux.Vars(r)["APP_ID"]
	requestCache := cache.GetCache()
	cacheKey := computeGetApiKeyScopesCacheKey(appId)
	if cachedValue := requestCache.Get(cacheKey); cachedValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedValue))
		return
	}
	scopes, err := h.service.GetScopes(r.Context(), appId)
	if err != nil {
		renderApiKeyScopeServiceError(w, err)
		return
	}
	response := make([]ApiKeyScopesResponse, 0, len(scopes))
	for _, scope := range scopes {
		entry := ApiKeyScopesResponse{
			ApiKeyID:   strconv.FormatInt(scope.ApiKeyID, 10),
			ChannelIDs: make([]string, 0, len(scope.ChannelIDs)),
			AllowedIps: make([]string, 0, len(scope.AllowedIps)),
		}
		for _, channelID := range scope.ChannelIDs {
			entry.ChannelIDs = append(entry.ChannelIDs, strconv.FormatInt(channelID, 10))
		}
		for _, prefix := range scope.AllowedIps {
			entry.AllowedIps = append(entry.AllowedIps, prefix.String())
		}
		response = append(response, entry)
	}
	marshaledResponse, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaledResponse)

	ttl := 60
	requestCache.Set(cacheKey, string(marshaledResponse), &ttl)
}

func (h *ApiKeyScopeHandler) SetApiKeyScopesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId := vars["APP_ID"]
	apiKeyID, err := strconv.ParseInt(vars["API_KEY_ID"], 10, 64)
	if err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "Invalid API key ID")
		return
	}
	var req struct {
		ChannelIDs []string `json:"channelIds"`
		AllowedIps []string `json:"allowedIps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	channelIDs := make([]int64, 0, len(req.ChannelIDs))
	for _, raw := range req.ChannelIDs {
		channelID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			handlers.RenderError(w, http.StatusBadRequest, "Invalid channel ID: "+raw)
			return
		}
		channelIDs = append(channelIDs, channelID)
	}
	if err := h.service.SetScopes(r.Context(), appId, apiKeyID, channelIDs, req.AllowedIps); err != nil {
		renderApiKeyScopeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)

	cache.GetCache().Delete(computeGetApiKeyScopesCacheKey(appId))
}
