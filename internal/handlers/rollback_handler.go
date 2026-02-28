package handlers

import (
	"encoding/json"
	"expo-open-ota/internal/branch"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/update"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func RollbackHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	branchName := vars["BRANCH"]
	platform := r.URL.Query().Get("platform")
	if platform == "" || (platform != "ios" && platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}
	if branchName == "" {
		log.Printf("[RequestID: %s] No branch provided", requestID)
		http.Error(w, "No branch provided", http.StatusBadRequest)
		return
	}
	expoAuth := helpers.GetExpoAuth(r)
	expoAccount, err := services.FetchExpoUserAccountInformations(expoAuth)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching expo account informations: %v", requestID, err)
		http.Error(w, "Error fetching expo account informations", http.StatusUnauthorized)
		return
	}
	if expoAccount == nil {
		log.Printf("[RequestID: %s] No expo account found", requestID)
		http.Error(w, "No expo account found", http.StatusUnauthorized)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
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
