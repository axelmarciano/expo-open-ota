package handlers

import (
	"encoding/json"
	"errors"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type UploadHandler struct {
	cliAuthService    *services.CliAuthService
	deploymentService *services.DeploymentService
}

func NewUploadHandler(cliAuthService *services.CliAuthService, deploymentService *services.DeploymentService) *UploadHandler {
	return &UploadHandler{
		cliAuthService:    cliAuthService,
		deploymentService: deploymentService,
	}
}

type FileNamesRequest struct {
	FileNames []string `json:"fileNames"`
	Message   string   `json:"message,omitempty"`
}

func (h *UploadHandler) MarkUpdateAsUploadedHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	appId := vars["APP_ID"]
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
	auth := helpers.GetAuth(r)
	err := h.cliAuthService.ValidateCliCredential(r.Context(), appId, auth)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating auth: %v", requestID, err)
		http.Error(w, "Error validating auth", http.StatusUnauthorized)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}
	updateId := r.URL.Query().Get("updateId")
	if updateId == "" {
		log.Printf("[RequestID: %s] No update id provided", requestID)
		http.Error(w, "No update id provided", http.StatusBadRequest)
		return
	}
	params := services.ProcessUpdateParams{
		RequestID:      requestID,
		AppID:          appId,
		BranchName:     branchName,
		Platform:       platform,
		RuntimeVersion: runtimeVersion,
		UpdateID:       updateId,
	}
	err = h.deploymentService.ProcessUploadedUpdate(r.Context(), params)
	if err != nil {
		if errors.Is(err, services.ErrUnauthorized) {
			http.Error(w, "Error validating auth", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, services.ErrInvalidUpdate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, services.ErrNoChangesDetected) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotAcceptable)

			response := map[string]string{
				"error": "You have already uploaded this update, no changes detected",
			}
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Any unexpected runtime/database systems error falls back to standard 500s
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *UploadHandler) RequestUploadLocalFileHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	appId := mux.Vars(r)["APP_ID"]

	auth := helpers.GetAuth(r)
	err := h.cliAuthService.ValidateCliCredential(r.Context(), appId, auth)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating auth: %v", requestID, err)
		http.Error(w, "Error validating auth", http.StatusUnauthorized)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		log.Printf("[RequestID: %s] No token provided", requestID)
		http.Error(w, "No token provided", http.StatusBadRequest)
		return
	}

	filePath, tokenAppId, err := bucket.ValidateUploadTokenAndResolveFilePath(token)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating upload token: %v", requestID, err)
		http.Error(w, "Error validating upload token", http.StatusBadRequest)
		return
	}

	fileName := filepath.Base(filePath)
	file, _, err := r.FormFile(fileName)
	if err != nil {
		log.Printf("[RequestID: %s] Error retrieving file from form: %v", requestID, err)
		http.Error(w, "Error retrieving file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	params := services.RequestLocalFileUploadParams{
		RequestID:  requestID,
		AppID:      appId,
		Token:      token,
		Body:       file,
		FilePath:   filePath,
		TokenAppID: tokenAppId,
	}

	if err := h.deploymentService.RequestUploadLocalFile(r.Context(), params); err != nil {
		if errors.Is(err, services.ErrInvalidBucketType) {
			http.Error(w, "Invalid bucket type", http.StatusInternalServerError)
			return
		}
		if errors.Is(err, services.ErrInvalidToken) {
			http.Error(w, "Error validating upload token", http.StatusBadRequest)
			return
		}
		if errors.Is(err, services.ErrTokenAppMismatch) {
			http.Error(w, "Upload token does not match this app", http.StatusForbidden)
			return
		}
		if errors.Is(err, services.ErrUploadFailed) {
			http.Error(w, "Error handling upload file", http.StatusInternalServerError)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UploadHandler) RequestUploadUrlHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	appId := vars["APP_ID"]
	branchName := vars["BRANCH"]
	if branchName == "" {
		log.Printf("[RequestID: %s] No branch provided", requestID)
		http.Error(w, "No branch provided", http.StatusBadRequest)
		return
	}

	auth := helpers.GetAuth(r)
	err := h.cliAuthService.ValidateCliCredential(r.Context(), appId, auth)
	if err != nil {
		log.Printf("[RequestID: %s] Error validating auth: %v", requestID, err)
		http.Error(w, "Error validating auth", http.StatusUnauthorized)
		return
	}

	platform := r.URL.Query().Get("platform")
	if platform != "" && (platform != "ios" && platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}
	commitHash := r.URL.Query().Get("commitHash")
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}

	var bodyReq FileNamesRequest
	if err := json.NewDecoder(r.Body).Decode(&bodyReq); err != nil {
		log.Printf("[RequestID: %s] Error decoding JSON body: %v", requestID, err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(bodyReq.FileNames) == 0 {
		log.Printf("[RequestID: %s] No file names provided", requestID)
		http.Error(w, "No file names provided", http.StatusBadRequest)
		return
	}

	params := services.RequestUploadURLParams{
		RequestID:      requestID,
		AppID:          appId,
		BranchName:     branchName,
		Platform:       platform,
		CommitHash:     commitHash,
		RuntimeVersion: runtimeVersion,
		FileNames:      bodyReq.FileNames,
		Message:        bodyReq.Message,
	}

	result, err := h.deploymentService.RequestUploadURLs(r.Context(), params)
	if err != nil {
		http.Error(w, "Internal server error processing payload URLs", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"updateId":       result.UpdateID,
		"uploadRequests": result.UploadRequests,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("expo-update-id", fmt.Sprintf("%d", result.UpdateID))
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[RequestID: %s] Error encoding response serialization: %v", requestID, err)
	}

}
