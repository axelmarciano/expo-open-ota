package assets

import (
	"expo-open-ota/internal/modules/bucket"
	"expo-open-ota/internal/modules/environments"
	"expo-open-ota/internal/modules/types"
	"expo-open-ota/internal/modules/update"
	"log"
	"mime"
	"net/http"
)

type AssetsRequest struct {
	Environment    string
	AssetName      string
	RuntimeVersion string
	Platform       string
	RequestID      string
}

type AssetsResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        []byte
	ContentType string
	URL         string
}

func getAssetMetadata(req AssetsRequest) (AssetsResponse, *types.BucketFile, error) {
	requestID := req.RequestID

	if !environments.ValidateEnvironment(req.Environment) {
		log.Printf("[RequestID: %s] Invalid environment: %s", requestID, req.Environment)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Invalid environment")}, nil, nil
	}

	if req.AssetName == "" {
		log.Printf("[RequestID: %s] No asset name provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No asset name provided")}, nil, nil
	}

	if req.Platform == "" || (req.Platform != "ios" && req.Platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, req.Platform)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Invalid platform")}, nil, nil
	}

	if req.RuntimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No runtime version provided")}, nil, nil
	}

	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion(req.Environment, req.RuntimeVersion)
	if err != nil || lastUpdate == nil {
		log.Printf("[RequestID: %s] No update found for runtimeVersion: %s", requestID, req.RuntimeVersion)
		return AssetsResponse{StatusCode: http.StatusNotFound, Body: []byte("No update found")}, nil, nil
	}

	metadata, err := update.GetMetadata(*lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting metadata")}, nil, nil
	}

	var platformMetadata types.PlatformMetadata
	switch req.Platform {
	case "android":
		platformMetadata = metadata.MetadataJSON.FileMetadata.Android
	case "ios":
		platformMetadata = metadata.MetadataJSON.FileMetadata.IOS
	default:
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Platform not supported")}, nil, nil
	}

	bundle := platformMetadata.Bundle
	isLaunchAsset := bundle == req.AssetName

	var assetMetadata types.Asset
	for _, asset := range platformMetadata.Assets {
		if asset.Path == req.AssetName {
			assetMetadata = asset
		}
	}

	resolvedBucket, err := bucket.GetBucket()
	if err != nil {
		log.Printf("[RequestID: %s] Error resolving bucket: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error resolving bucket")}, nil, nil
	}

	asset, err := resolvedBucket.GetFile(*lastUpdate, req.AssetName)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting asset: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting asset")}, nil, nil
	}

	var contentType string
	if isLaunchAsset {
		contentType = "application/javascript"
	} else {
		contentType = mime.TypeByExtension("." + string(assetMetadata.Ext))
	}

	headers := map[string]string{
		"expo-protocol-version": "1",
		"expo-sfv-version":      "0",
		"Cache-Control":         "public, max-age=31536000",
		"Content-Type":          contentType,
	}

	return AssetsResponse{
		StatusCode:  http.StatusOK,
		Headers:     headers,
		ContentType: contentType,
	}, &asset, nil
}

func HandleAssetsWithFile(req AssetsRequest) (AssetsResponse, error) {
	resp, asset, err := getAssetMetadata(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != 200 {
		return AssetsResponse{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
		}, nil
	}

	if asset == nil {
		log.Printf("[RequestID: %s] Resolved file is nil", req.RequestID)
		return AssetsResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       []byte("Resolved file is nil"),
		}, nil
	}

	buffer, err := bucket.ConvertReadCloserToBytes(asset.Reader)
	defer asset.Reader.Close()
	if err != nil {
		log.Printf("[RequestID: %s] Error converting asset to buffer: %v", req.RequestID, err)
		return AssetsResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       []byte("Error converting asset to buffer"),
		}, err
	}

	resp.Body = buffer
	return resp, nil
}

func HandleAssetsWithURL(req AssetsRequest, domain string) (AssetsResponse, error) {
	resp, _, err := getAssetMetadata(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != 200 {
		return AssetsResponse{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
		}, nil
	}
	resp.URL = domain + "/" + req.Environment + "/" + req.AssetName
	return resp, nil
}
