package handlers

import (
	"encoding/json"
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
