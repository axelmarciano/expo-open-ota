package handlers

import (
	"encoding/json"
	"errors"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	types2 "expo-open-ota/internal/types"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type RepublishHandler struct {
	cliAuthService    *services.CliAuthService
	deploymentService *services.DeploymentService
}

func NewRepublishHandler(cliAuthService *services.CliAuthService, deploymentService *services.DeploymentService) *RepublishHandler {
	return &RepublishHandler{
		cliAuthService:    cliAuthService,
		deploymentService: deploymentService,
	}
}

func (h *RepublishHandler) HandleRepublish(w http.ResponseWriter, r *http.Request) {
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
	err := h.cliAuthService.ValidateCliCredential(r.Context(), appId, auth, branchName, helpers.ClientIP(r))
	if err != nil {
		log.Printf("[RequestID: %s] Error validating auth: %v", requestID, err)
		RenderCliAuthError(w, err)
		return
	}
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}
	commitHash := r.URL.Query().Get("commitHash")
	updateId := r.URL.Query().Get("updateId")
	if updateId == "" {
		log.Printf("[RequestID: %s] No updateId provided", requestID)
		http.Error(w, "No updateId provided", http.StatusBadRequest)
		return
	}
	previousUpdate := &types2.Update{
		AppId:          appId,
		Branch:         branchName,
		RuntimeVersion: runtimeVersion,
		UpdateId:       updateId,
	}
	newUpdate, err := h.deploymentService.RepublishUpdate(r.Context(), previousUpdate, platform, commitHash)
	if err != nil {
		if errors.Is(err, services.ErrActiveRolloutBlocksPublish) {
			log.Printf("[RequestID: %s] Republish blocked by active rollout: %v", requestID, err)
			http.Error(w, activeRolloutConflictMessage, http.StatusConflict)
			return
		}
		var rErr *services.RepublishError
		if errors.As(err, &rErr) {
			http.Error(w, rErr.Message, rErr.Status)
			return
		}
		http.Error(w, "Error republishing update", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUpdate)
}
