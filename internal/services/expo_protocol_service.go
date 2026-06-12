package services

import (
	"bytes"
	"context"
	"encoding/json"
	"expo-open-ota/internal/assets"
	cdn2 "expo-open-ota/internal/cdn"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/keyStore"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
)

type ExpoProtocolService struct {
	appRepo       AppRepository
	channelRepo   ChannelRepository
	updateRepo    UpdateRepository
	updateService *UpdateService
}

type ManifestRequestParams struct {
	RequestID             string
	AppID                 string
	ChannelName           string
	Platform              string
	RuntimeVersion        string
	ProtocolVersion       int64
	ClientID              string
	CurrentUpdateID       string
	ExpoFatalError        string
	RecentFailedUpdateIDs string
}

type ManifestResult struct {
	Update     *types.Update
	BranchName string
	UpdateType types.UpdateType
}

type ExpoProtocolError struct {
	StatusCode int
	Message    string
}

type AssetResolutionParams struct {
	RequestID             string
	AppID                 string
	ChannelName           string
	AssetName             string
	RuntimeVersion        string
	Platform              string
	PreventCDNRedirection bool
}

type ExpoAssetError struct {
	StatusCode int
	Message    string
}

type ExpoAssetResult struct {
	RedirectToURL string
	Body          []byte
	ContentType   string
	Headers       map[string]string
	StatusCode    int
}

func (e *ExpoProtocolError) Error() string { return e.Message }

func (e *ExpoAssetError) Error() string { return e.Message }

func NewExpoProtocolService(appRepo AppRepository, channelRepo ChannelRepository, updateRepo UpdateRepository, updateService *UpdateService) *ExpoProtocolService {
	return &ExpoProtocolService{
		appRepo:       appRepo,
		channelRepo:   channelRepo,
		updateRepo:    updateRepo,
		updateService: updateService,
	}
}

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

func (s *ExpoProtocolService) signDirectiveOrManifest(ctx context.Context, appId string, content interface{}, expectSignatureHeader string) (string, error) {
	if expectSignatureHeader == "" {
		return "", nil
	}
	appConfig, err := s.appRepo.GetAppByID(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("failed to fetch app config for app ID '%s': %w", appId, err)
	}
	privateKey := keyStore.GetPrivateExpoKey(appConfig)
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

func (s *ExpoProtocolService) PutUpdateInResponse(w http.ResponseWriter, r *http.Request, appId string, lastUpdate types.Update, platform string, protocolVersion int64, requestID string) {
	currentUpdateId := r.Header.Get("expo-current-update-id")
	metadata, err := update2.GetMetadata(lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		http.Error(w, "Error getting metadata", http.StatusInternalServerError)
		return
	}

	if currentUpdateId != "" && currentUpdateId == crypto.ConvertSHA256HashToUUID(metadata.ID) && protocolVersion == 1 {
		s.PutNoUpdateAvailableInResponse(w, r, appId, lastUpdate.RuntimeVersion, protocolVersion, requestID)
		return
	}
	manifest, err := update2.ComposeUpdateManifest(&metadata, lastUpdate, platform)
	if err != nil {
		log.Printf("[RequestID: %s] Error composing manifest: %v", requestID, err)
		http.Error(w, "Error composing manifest", http.StatusInternalServerError)
		return
	}
	if currentUpdateId != "" {
		metrics.TrackUpdateDownload(appId, platform, lastUpdate.RuntimeVersion, lastUpdate.Branch, manifest.Id, "update")
	}
	w.Header().Set("expo-manifest-filters", `branch="`+lastUpdate.Branch+`"`)
	s.PutResponse(w, r, appId, manifest, "manifest", lastUpdate.RuntimeVersion, protocolVersion, requestID)
}

func (s *ExpoProtocolService) PutResponse(w http.ResponseWriter, r *http.Request, appId string, content interface{}, fieldName string, runtimeVersion string, protocolVersion int64, requestID string) {
	signedHash, err := s.signDirectiveOrManifest(r.Context(), appId, content, r.Header.Get("expo-expect-signature"))
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

func (s *ExpoProtocolService) PutRollbackInResponse(w http.ResponseWriter, r *http.Request, appId string, lastUpdate types.Update, platform string, protocolVersion int64, requestID string) {
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
		s.PutNoUpdateAvailableInResponse(w, r, appId, lastUpdate.RuntimeVersion, protocolVersion, requestID)
		return
	}

	normalizedTime := helpers.NormalizeTimestamp(int64(lastUpdate.CreatedAt))
	commitTime := normalizedTime.Format("2006-01-02T15:04:05.000Z")
	directive, err := update2.CreateRollbackDirective(lastUpdate, commitTime)
	if err != nil {
		log.Printf("[RequestID: %s] Error creating rollback directive: %v", requestID, err)
		http.Error(w, "Error creating rollback directive", http.StatusInternalServerError)
		return
	}
	metrics.TrackUpdateDownload(appId, platform, lastUpdate.RuntimeVersion, lastUpdate.Branch, lastUpdate.UpdateId, "rollback")
	s.PutResponse(w, r, appId, directive, "directive", lastUpdate.RuntimeVersion, protocolVersion, requestID)
}

func (s *ExpoProtocolService) PutNoUpdateAvailableInResponse(w http.ResponseWriter, r *http.Request, appId string, runtimeVersion string, protocolVersion int64, requestID string) {
	if protocolVersion == 0 {
		http.Error(w, "NoUpdateAvailable directive not available in protocol version 0", http.StatusNoContent)
		return
	}
	directive := update2.CreateNoUpdateAvailableDirective()
	s.PutResponse(w, r, appId, directive, "directive", runtimeVersion, protocolVersion, requestID)
}

func (s *ExpoProtocolService) ResolveManifestBundle(ctx context.Context, params ManifestRequestParams) (ManifestResult, error) {
	// [Stateless mode] Reject unknown app ids at the edge with a clean 404 — otherwise
	// downstream services.FetchExpoChannelMapping → GetExpoAccessToken
	// returns an empty token for the unknown id and we end up POSTing to
	// api.expo.dev with `Bearer ` (no token), surfacing the upstream 401
	// as an opaque 500 to the client.
	if _, err := s.appRepo.GetAppByID(ctx, params.AppID); err != nil {
		log.Printf("[RequestID: %s] Unknown app id %q", params.RequestID, params.AppID)
		return ManifestResult{}, &ExpoProtocolError{StatusCode: http.StatusNotFound, Message: "Unknown app id"}
	}

	branchMap, err := s.channelRepo.GetChannelBranchMapping(ctx, params.AppID, params.ChannelName)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching channel mapping: %v", params.RequestID, err)
		return ManifestResult{}, &ExpoProtocolError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("Error fetching channel mapping: %v", err)}
	}
	if branchMap == nil {
		log.Printf("[RequestID: %s] No branch mapping found for channel: %s", params.RequestID, params.ChannelName)
		return ManifestResult{}, &ExpoProtocolError{StatusCode: http.StatusNotFound, Message: "No branch mapping found"}
	}

	branchName := branchMap.BranchName

	if params.ExpoFatalError != "" {
		if params.CurrentUpdateID != "" {
			metrics.TrackUpdateErrorUsers(params.AppID, params.ClientID, params.Platform, params.RuntimeVersion, branchName, params.CurrentUpdateID)
		} else if params.RecentFailedUpdateIDs != "" {
			metrics.TrackUpdateErrorUsers(params.AppID, params.ClientID, params.Platform, params.RuntimeVersion, branchName, params.RecentFailedUpdateIDs)
		}
	}
	metrics.TrackActiveUser(params.AppID, params.ClientID, params.Platform, params.RuntimeVersion, branchName, params.CurrentUpdateID)

	lastUpdate, err := s.updateService.GetLatestUpdate(ctx, params.AppID, branchName, params.RuntimeVersion, params.Platform)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting latest update: %v", params.RequestID, err)
		return ManifestResult{}, &ExpoProtocolError{StatusCode: http.StatusInternalServerError, Message: "Error getting latest update"}
	}
	if lastUpdate == nil {
		return ManifestResult{
			Update:     nil,
			BranchName: branchName,
		}, nil
	}
	updateType, err := s.updateRepo.GetUpdateType(ctx, *lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error determining update type: %v", params.RequestID, err)
		return ManifestResult{}, &ExpoProtocolError{StatusCode: http.StatusInternalServerError, Message: "Error determining update type"}
	}

	return ManifestResult{Update: lastUpdate, BranchName: branchName, UpdateType: updateType}, nil
}

func (s *ExpoProtocolService) ResolveAssetBundle(ctx context.Context, params AssetResolutionParams) (*ExpoAssetResult, error) {
	// [Stateless mode] Same edge check as ManifestHandler — reject unknown ids with 404
	// rather than letting them flow into FetchExpoChannelMapping and
	// surfacing the upstream 401 as a 500.
	if _, err := s.appRepo.GetAppByID(ctx, params.AppID); err != nil {
		log.Printf("[RequestID: %s] Unknown app id %q", params.RequestID, params.AppID)
		return &ExpoAssetResult{}, &ExpoProtocolError{StatusCode: http.StatusNotFound, Message: "Unknown app id"}
	}

	branchMap, err := s.channelRepo.GetChannelBranchMapping(ctx, params.AppID, params.ChannelName)
	if err != nil {
		log.Printf("[RequestID: %s] Error fetching channel mapping: %v", params.RequestID, err)
		return &ExpoAssetResult{}, &ExpoProtocolError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("Error fetching channel mapping: %v", err)}
	}
	if branchMap == nil {
		log.Printf("[RequestID: %s] No branch mapping found for channel: %s", params.RequestID, params.ChannelName)
		return &ExpoAssetResult{}, &ExpoProtocolError{StatusCode: http.StatusNotFound, Message: "No branch mapping found"}
	}
	lastUpdate, err := s.updateService.GetLatestUpdate(ctx, params.AppID, branchMap.BranchName, params.RuntimeVersion, params.Platform)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting latest update: %v", params.RequestID, err)
		return &ExpoAssetResult{}, &ExpoProtocolError{StatusCode: http.StatusInternalServerError, Message: "Error getting latest update"}
	}

	branchName := branchMap.BranchName

	req := assets.AssetsRequest{
		AppId:          params.AppID,
		Branch:         branchName,
		AssetName:      params.AssetName,
		RuntimeVersion: params.RuntimeVersion,
		Platform:       params.Platform,
		Update:         lastUpdate,
		RequestID:      params.RequestID,
	}

	cdn := cdn2.GetCDN()

	if cdn == nil || params.PreventCDNRedirection {
		resp, err := assets.HandleAssetsWithFile(req)
		if err != nil {
			return nil, &ExpoAssetError{StatusCode: http.StatusInternalServerError, Message: "Internal Server Error"}
		}

		return &ExpoAssetResult{
			Body:        resp.Body,
			ContentType: resp.ContentType,
			Headers:     resp.Headers,
			StatusCode:  resp.StatusCode,
		}, nil
	}

	resp, err := assets.HandleAssetsWithURL(req, cdn)
	if err != nil {
		return nil, &ExpoAssetError{StatusCode: http.StatusInternalServerError, Message: "Internal Server Error"}
	}

	return &ExpoAssetResult{
		RedirectToURL: resp.URL,
	}, nil
}
