package handlers

import (
	"encoding/json"
	"expo-open-ota/internal/auth"
	"expo-open-ota/internal/branch"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/update"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func RollbackHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	branchName, _ := url.PathUnescape(vars["BRANCH"])
	platform := r.URL.Query().Get("platform")
	if platform == "" || (platform != "ios" && platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}
	if err := helpers.ValidateResourceName(branchName, "branch"); err != nil {
		log.Printf("[RequestID: %s] Invalid branch name: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	eoasAuth := helpers.GetEoasAuth(r)
	err := auth.ValidateEOASAuth(&eoasAuth)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating auth: %v", requestID, err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if err := helpers.ValidateResourceName(runtimeVersion, "runtimeVersion"); err != nil {
		log.Printf("[RequestID: %s] Invalid runtime version: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	errUpsert := branch.UpsertBranch(branchName)
	if errUpsert != nil {
		log.Printf("[RequestID: %s] Error upserting branch: %v", requestID, errUpsert)
		http.Error(w, "Error upserting branch", http.StatusInternalServerError)
		return
	}
	commitHash := r.URL.Query().Get("commitHash")
	rollback, err := update.CreateRollback(platform, commitHash, runtimeVersion, branchName)
	if err != nil {
		log.Printf("[RequestID: %s] Error creating rollback: %v", requestID, err)
		http.Error(w, "Error creating rollback", http.StatusInternalServerError)
		return
	}
	log.Printf("[RequestID: %s] Rollback created: %s", requestID, rollback.UpdateId)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rollback)
}
