package handlers

import (
	"encoding/json"
	"expo-open-ota/config"
	"expo-open-ota/internal/modules/bucket"
	"expo-open-ota/internal/modules/environments"
	"expo-open-ota/internal/modules/helpers"
	"expo-open-ota/internal/services"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

type FileNamesRequest struct {
	FileNames []string `json:"fileNames"`
}

func RequestUploadLocalFileHandler(w http.ResponseWriter, r *http.Request) {
	bucketType := bucket.ResolveBucketType()
	if bucketType != bucket.LocalBucketType {
		log.Printf("Invalid bucket type: %s", bucketType)
		http.Error(w, "Invalid bucket type", http.StatusInternalServerError)
	}
	requestID := uuid.New().String()
	bearerToken, err := helpers.GetBearerToken(r)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting bearer token: %v", requestID, err)
		http.Error(w, "Error getting bearer token", http.StatusInternalServerError)
		return
	}
	expoAccount, err := services.FetchExpoUserAccountInformations(bearerToken)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching expo account informations: %v", requestID, err)
		http.Error(w, "Error fetching expo account informations", http.StatusInternalServerError)
		return
	}
	if expoAccount == nil {
		log.Printf("[RequestID: %s] No expo account found", requestID)
		http.Error(w, "No expo account found", http.StatusUnauthorized)
		return
	}
	currentExpoUsername := config.GetEnv("EXPO_USERNAME")
	if expoAccount.Username != currentExpoUsername {
		log.Printf("[RequestID: %s] Invalid expo account", requestID)
		http.Error(w, "Invalid expo account", http.StatusUnauthorized)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		log.Printf("[RequestID: %s] No token provided", requestID)
		http.Error(w, "No token provided", http.StatusBadRequest)
		return
	}
	filePath, err := bucket.ValidateUploadTokenAndResolveFilePath(token)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating upload token: %v", requestID, err)
		http.Error(w, "Error validating upload token", http.StatusBadRequest)
		return
	}
	if r.Body == nil {
		log.Printf("[RequestID: %s] Empty request body", requestID)
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fileName := filepath.Base(filePath)

	file, _, err := r.FormFile(fileName)
	if err != nil {
		log.Printf("[RequestID: %s] Error retrieving file from form: %v", requestID, err)
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}

	success, err := bucket.HandleUploadFile(filePath, file)
	if err != nil {
		log.Printf("[RequestID: %s] Error handling upload file: %v", requestID, err)
		http.Error(w, "Error handling upload file", http.StatusInternalServerError)
		return
	}
	if !success {
		log.Printf("[RequestID: %s] Error handling upload file", requestID)
		http.Error(w, "Error handling upload file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RequestUploadUrlHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	environment := vars["ENVIRONMENT"]
	bearerToken, err := helpers.GetBearerToken(r)

	if err != nil {
		log.Printf("[RequestID: %s] Error getting bearer token: %v", requestID, err)
		http.Error(w, "Error getting bearer token", http.StatusInternalServerError)
		return
	}
	expoAccount, err := services.FetchExpoUserAccountInformations(bearerToken)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching expo account informations: %v", requestID, err)
		http.Error(w, "Error fetching expo account informations", http.StatusInternalServerError)
		return
	}
	if expoAccount == nil {
		log.Printf("[RequestID: %s] No expo account found", requestID)
		http.Error(w, "No expo account found", http.StatusUnauthorized)
		return
	}
	currentExpoUsername := config.GetEnv("EXPO_USERNAME")
	if expoAccount.Username != currentExpoUsername {
		log.Printf("[RequestID: %s] Invalid expo account", requestID)
		http.Error(w, "Invalid expo account", http.StatusUnauthorized)
		return
	}
	if bearerToken == "" {
		log.Printf("[RequestID: %s] No bearer token provided", requestID)
		http.Error(w, "No bearer token provided", http.StatusUnauthorized)
		return
	}
	if !environments.ValidateEnvironment(environment) {
		log.Printf("[RequestID: %s] Invalid environment: %s", requestID, environment)
		http.Error(w, "Invalid environment", http.StatusBadRequest)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}
	var request FileNamesRequest
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("[RequestID: %s] Error decoding JSON body: %v", requestID, err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(request.FileNames) == 0 {
		log.Printf("[RequestID: %s] No file names provided", requestID)
		http.Error(w, "No file names provided", http.StatusBadRequest)
		return
	}
	fileNamesArray := request.FileNames
	updateId := time.Now().UnixNano() / int64(time.Millisecond)
	fmt.Println("currentTimeMs", updateId)
	w.Header().Set("Content-Type", "application/json")
	updateRequests, err := bucket.RequestUploadUrlsForFileUpdates(environment, runtimeVersion, fmt.Sprintf("%d", updateId), fileNamesArray)
	if err != nil {
		log.Printf("[RequestID: %s] Error requesting upload urls: %v", requestID, err)
		http.Error(w, "Error requesting upload urls", http.StatusInternalServerError)
		return
	}
	// Write json response (its not in helpers)
	jsonResponse, err := json.Marshal(updateRequests)
	if err != nil {
		log.Printf("[RequestID: %s] Error marshalling response: %v", requestID, err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, errWrite := w.Write(jsonResponse)
	if errWrite != nil {
		log.Printf("[RequestID: %s] Error writing response: %v", requestID, errWrite)
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		return
	}
}
