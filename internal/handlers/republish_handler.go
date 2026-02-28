package handlers

import (
	"encoding/json"
	"expo-open-ota/internal/auth"
	"expo-open-ota/internal/branch"
	"expo-open-ota/internal/helpers"
	types2 "expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func RepublishHandler(w http.ResponseWriter, r *http.Request) {
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
	err = branch.UpsertBranch(branchName)
	if err != nil {
		log.Printf("[RequestID: %s] Error upserting branch: %v", requestID, err)
		http.Error(w, "Error upserting branch", http.StatusInternalServerError)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if err := helpers.ValidateResourceName(runtimeVersion, "runtimeVersion"); err != nil {
		log.Printf("[RequestID: %s] Invalid runtime version: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	commitHash := r.URL.Query().Get("commitHash")
	updateId := r.URL.Query().Get("updateId")
	if updateId == "" {
		log.Printf("[RequestID: %s] No updateId provided", requestID)
		http.Error(w, "No updateId provided", http.StatusBadRequest)
		return
	}
	update, err := update2.GetUpdate(branchName, runtimeVersion, updateId)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting update: %v", requestID, err)
		http.Error(w, "Error getting update", http.StatusBadRequest)
		return
	}
	if update == nil {
		log.Printf("[RequestID: %s] No update found for runtimeVersion: %s in branch: %s", requestID, runtimeVersion, branchName)
		http.Error(w, "No update found", http.StatusNotFound)
		return
	}
	updateType := update2.GetUpdateType(*update)
	if updateType != types2.NormalUpdate {
		log.Printf("[RequestID: %s] Update type is not normal update", requestID)
		http.Error(w, "Update type is not normal update", http.StatusBadRequest)
		return
	}
	storedMetadata, err := update2.RetrieveUpdateStoredMetadata(*update)
	if err != nil {
		log.Printf("[RequestID: %s] Error retrieving update commit hash and platform: %v", requestID, err)
		http.Error(w, "Error retrieving update commit hash and platform", http.StatusInternalServerError)
		return
	}
	if storedMetadata == nil {
		log.Printf("[RequestID: %s] No stored metadata found for update: %s", requestID, updateId)
		http.Error(w, "No stored metadata found for update", http.StatusNotFound)
		return
	}
	isValid := update2.IsUpdateValid(*update)
	if !isValid {
		log.Printf("[RequestID: %s] Update is not valid", requestID)
		http.Error(w, "Update is not valid", http.StatusBadRequest)
		return
	}
	if storedMetadata.Platform != platform {
		log.Printf("[RequestID: %s] Update platform mismatch: %s != %s", requestID, storedMetadata.Platform, platform)
		http.Error(w, "Update platform mismatch", http.StatusBadRequest)
		return
	}
	newUpdate, err := update2.RepublishUpdate(update, platform, commitHash)
	if err != nil {
		log.Printf("[RequestID: %s] Error republishing update: %v", requestID, err)
		http.Error(w, "Error republishing update", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUpdate)
}
