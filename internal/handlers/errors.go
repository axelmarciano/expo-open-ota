package handlers

import (
	"encoding/json"
	"errors"
	"expo-open-ota/internal/services"
	"net/http"
)

type APIError struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// RenderError enforces the structured RFC 7807 error format
func RenderError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}

// RenderCliAuthError distinguishes a credential that failed to authenticate
// (401, generic message so nothing leaks about why) from one that
// authenticated but is blocked by per-key access restrictions (403, with the
// reason so the CLI user knows what to fix).
func RenderCliAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrCliAccessDenied) {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Error(w, "Error validating auth", http.StatusUnauthorized)
}
