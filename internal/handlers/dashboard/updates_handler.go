package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/validation"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const (
	defaultUpdateFeedLimit = 50
	maxUpdateFeedLimit     = 100
)

type updateFeedCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	BranchID  int64     `json:"branchId"`
	UpdateID  int64     `json:"updateId"`
}

func parseUpdateFeedDate(raw string, endOfDay bool) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

func decodeUpdateFeedCursor(raw string) (*updateFeedCursor, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var cursor updateFeedCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}

func encodeUpdateFeedCursor(item types.UpdateFeedItem) string {
	encoded, _ := json.Marshal(updateFeedCursor{
		CreatedAt: item.FeedCreatedAt,
		BranchID:  item.BranchID,
		UpdateID:  mustParseUpdateID(item.UpdateId),
	})
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func mustParseUpdateID(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

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
		var valErr *validation.Error
		if errors.As(err, &valErr) {
			handlers.RenderError(w, http.StatusBadRequest, valErr.Error())
			return
		}
		handlers.RenderError(w, http.StatusBadRequest, "An internal error occurred while fetching update details.")
		return
	}
	updatesResponse := types.UpdateDetails{
		UpdateUUID:        update.UpdateUUID,
		UpdateId:          update.UpdateId,
		CreatedAt:         update.CreatedAt,
		CommitHash:        update.CommitHash,
		Platform:          update.Platform,
		Message:           update.Message,
		Type:              update.Type,
		ExpoConfig:        update.ExpoConfig,
		RolloutPercentage: update.RolloutPercentage,
		ControlUpdateId:   update.ControlUpdateId,
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
		var valErr *validation.Error
		if errors.As(err, &valErr) {
			handlers.RenderError(w, http.StatusBadRequest, valErr.Error())
			return
		}
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

func (h *UpdateHandler) GetUpdateFeedHandler(w http.ResponseWriter, r *http.Request) {
	appId := mux.Vars(r)["APP_ID"]
	params := r.URL.Query()
	limit := defaultUpdateFeedLimit
	if rawLimit := params.Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > maxUpdateFeedLimit {
			handlers.RenderError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	from, err := parseUpdateFeedDate(params.Get("from"), false)
	if err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "from must be an RFC3339 timestamp or YYYY-MM-DD date")
		return
	}
	to, err := parseUpdateFeedDate(params.Get("to"), true)
	if err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "to must be an RFC3339 timestamp or YYYY-MM-DD date")
		return
	}
	cursor, err := decodeUpdateFeedCursor(params.Get("cursor"))
	if err != nil {
		handlers.RenderError(w, http.StatusBadRequest, "cursor is invalid")
		return
	}
	query := types.UpdateFeedQuery{
		Branch:         params.Get("branch"),
		RuntimeVersion: params.Get("runtimeVersion"),
		Platform:       params.Get("platform"),
		UpdateUUID:     params.Get("uuid"),
		PublishGroup:   params.Get("groupId"),
		CommitHash:     params.Get("commitHash"),
		From:           from,
		To:             to,
		Limit:          limit + 1,
	}
	if cursor != nil {
		query.CursorCreatedAt = &cursor.CreatedAt
		query.CursorBranchID = cursor.BranchID
		query.CursorUpdateID = cursor.UpdateID
	}
	updates, err := h.updateService.GetUpdateFeed(r.Context(), appId, query)
	if err != nil {
		handlers.RenderError(w, http.StatusInternalServerError, "An internal error occurred while fetching the update feed.")
		return
	}
	if updates == nil {
		updates = []types.UpdateFeedItem{}
	}
	page := types.UpdateFeedPage{Items: updates}
	if len(updates) > limit {
		page.Items = updates[:limit]
		page.NextCursor = encodeUpdateFeedCursor(page.Items[len(page.Items)-1])
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(page)
}
