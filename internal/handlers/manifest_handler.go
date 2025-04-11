package handlers

import (
	"bytes"
	"encoding/json"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/keyStore"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/update"
	"fmt"
	"github.com/google/uuid"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
)

func createMultipartResponse(headers map[string][]string, jsonContent interface{}) (*multipart.Writer, *bytes.Buffer, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	field, err := writer.CreatePart(headers)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating multipart field: %w", err)
	}
	contentJSON, err := json.Marshal(jsonContent)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling JSON: %w", err)
	}
	if _, err := field.Write(contentJSON); err != nil {
		return nil, nil, fmt.Errorf("error writing JSON content: %w", err)
	}
	return writer, &buf, nil
}

func signDirectiveOrManifest(content interface{}, expectSignatureHeader string) (string, error) {
	if expectSignatureHeader == "" {
		return "", nil
	}
	privateKey := keyStore.GetPrivateExpoKey()
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("error stringifying content: %w", err)
	}
	signedHash, err := crypto.SignRSASHA256(string(contentJSON), privateKey)
	if err != nil {
		return "", fmt.Errorf("error signing content hash: %w", err)
	}
	return signedHash, nil
}

func writeResponse(w http.ResponseWriter, writer *multipart.Writer, buf *bytes.Buffer, protocolVersion int64, runtimeVersion string, requestID string) {
	w.Header().Set("expo-protocol-version", strconv.FormatInt(protocolVersion, 10))
	w.Header().Set("expo-sfv-version", "0")
	w.Header().Set("cache-control", "private, max-age=0")
	w.Header().Set("content-type", "multipart/mixed; boundary="+writer.Boundary())
	if err := writer.Close(); err != nil {
		log.Printf("[RequestID: %s] Error closing multipart writer: %v", requestID, err)
		http.Error(w, "Error closing multipart writer", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("[RequestID: %s] Error writing response: %v", requestID, err)
	}
}

func putResponse(w http.ResponseWriter, r *http.Request, content interface{}, fieldName string, runtimeVersion string, protocolVersion int64, requestID string) {
	signedHash, err := signDirectiveOrManifest(content, r.Header.Get("expo-expect-signature"))
	if err != nil {
		log.Printf("[RequestID: %s] Error signing content: %v", requestID, err)
		http.Error(w, "Error signing content", http.StatusInternalServerError)
		return
	}
	headers := map[string][]string{
		"Content-Disposition": {fmt.Sprintf("form-data; name=\"%s\"", fieldName)},
		"Content-Type":        {"application/json"},
		"content-type":        {"application/json; charset=utf-8"},
	}
	if signedHash != "" {
		headers["expo-signature"] = []string{fmt.Sprintf("sig=\"%s\", keyid=\"main\"", signedHash)}
	}
	writer, buf, err := createMultipartResponse(headers, content)
	if err != nil {
		log.Printf("[RequestID: %s] Error creating multipart response: %v", requestID, err)
		http.Error(w, "Error creating multipart response", http.StatusInternalServerError)
		return
	}
	writeResponse(w, writer, buf, protocolVersion, runtimeVersion, requestID)
}

func putUpdateInResponse(w http.ResponseWriter, r *http.Request, lastUpdate types.Update, platform string, protocolVersion int64, requestID string) {
	currentUpdateId := r.Header.Get("expo-current-update-id")
	metadata, err := update.GetMetadata(lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		http.Error(w, "Error getting metadata", http.StatusInternalServerError)
		return
	}
	if currentUpdateId != "" && currentUpdateId == crypto.ConvertSHA256HashToUUID(metadata.ID) && protocolVersion == 1 {
		putNoUpdateAvailableInResponse(w, r, lastUpdate.RuntimeVersion, protocolVersion, requestID)
		return
	}
	manifest, err := update.ComposeUpdateManifest(&metadata, lastUpdate, platform)
	if err != nil {
		log.Printf("[RequestID: %s] Error composing manifest: %v", requestID, err)
		http.Error(w, "Error composing manifest", http.StatusInternalServerError)
		return
	}
	channelOverride := helpers.GetChannelOverride(r.Header)
	if channelOverride != "" {
		update.AppendChannelOverrideToAsset(&manifest, channelOverride)
	}
	metrics.TrackUpdateDownload(platform, lastUpdate.RuntimeVersion, lastUpdate.Branch, metadata.ID, "update")
	putResponse(w, r, manifest, "manifest", lastUpdate.RuntimeVersion, protocolVersion, requestID)
}

func putRollbackInResponse(w http.ResponseWriter, r *http.Request, lastUpdate types.Update, platform string, protocolVersion int64, requestID string) {
	if protocolVersion == 0 {
		http.Error(w, "Rollback not supported in protocol version 0", http.StatusBadRequest)
		return
	}
	embeddedUpdateId := r.Header.Get("expo-embedded-update-id")
	if embeddedUpdateId == "" {
		http.Error(w, "No embedded update id provided", http.StatusBadRequest)
		return
	}
	currentUpdateId := r.Header.Get("expo-current-update-id")
	if currentUpdateId != "" && currentUpdateId == embeddedUpdateId {
		putNoUpdateAvailableInResponse(w, r, lastUpdate.RuntimeVersion, protocolVersion, requestID)
		return
	}
	directive, err := update.CreateRollbackDirective(lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error creating rollback directive: %v", requestID, err)
		http.Error(w, "Error creating rollback directive", http.StatusInternalServerError)
		return
	}
	metrics.TrackUpdateDownload(platform, lastUpdate.RuntimeVersion, lastUpdate.Branch, lastUpdate.UpdateId, "rollback")
	putResponse(w, r, directive, "directive", lastUpdate.RuntimeVersion, protocolVersion, requestID)
}

func putNoUpdateAvailableInResponse(w http.ResponseWriter, r *http.Request, runtimeVersion string, protocolVersion int64, requestID string) {
	if protocolVersion == 0 {
		http.Error(w, "NoUpdateAvailable directive not available in protocol version 0", http.StatusNoContent)
		return
	}
	directive := update.CreateNoUpdateAvailableDirective()
	putResponse(w, r, directive, "directive", runtimeVersion, protocolVersion, requestID)
}

func ManifestHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()

	channelName := helpers.ResolveExpoChannel(r.Header)
	if channelName == "" {
		log.Printf("[RequestID: %s] No channel name provided", requestID)
		http.Error(w, "No channel name provided", http.StatusBadRequest)
		return
	}
	branchMap, err := services.FetchExpoChannelMapping(channelName)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching channel mapping: %v", requestID, err)
		http.Error(w, "Error fetching channel mapping", http.StatusInternalServerError)
		return
	}
	if branchMap == nil {
		log.Printf("[RequestID: %s] No branch mapping found for channel: %s", requestID, channelName)
		http.Error(w, "No branch mapping found", http.StatusNotFound)
		return
	}
	branch := branchMap.BranchName
	protocolVersion, err := strconv.ParseInt(r.Header.Get("expo-protocol-version"), 10, 64)
	if err != nil {
		log.Printf("[RequestID: %s] Invalid protocol version: %v", requestID, err)
		http.Error(w, "Invalid protocol version", http.StatusBadRequest)
		return
	}
	platform := r.Header.Get("expo-platform")
	if platform == "" {
		platform = r.URL.Query().Get("platform")
	}
	if platform == "" || (platform != "ios" && platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}
	runtimeVersion := r.Header.Get("expo-runtime-version")
	if runtimeVersion == "" {
		runtimeVersion = r.URL.Query().Get("runtimeVersion")
	}
	clientId := r.Header.Get("EAS-Client-ID")
	currentUpdateId := r.Header.Get("expo-current-update-id")
	metrics.TrackActiveUser(clientId, platform, runtimeVersion, branch, currentUpdateId)
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}
	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion(branch, runtimeVersion)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting latest update: %v", requestID, err)
		http.Error(w, "Error getting latest update", http.StatusInternalServerError)
		return
	}
	if lastUpdate == nil {
		log.Printf("[RequestID: %s] No update found for runtimeVersion: %s in branch: %s", requestID, runtimeVersion, branch)
		putNoUpdateAvailableInResponse(w, r, runtimeVersion, protocolVersion, requestID)
		return
	}

	updateType := update.GetUpdateType(*lastUpdate)
	if updateType == types.NormalUpdate {
		putUpdateInResponse(w, r, *lastUpdate, platform, protocolVersion, requestID)
	} else {
		putRollbackInResponse(w, r, *lastUpdate, platform, protocolVersion, requestID)
	}
}
