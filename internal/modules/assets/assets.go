package assets

// We use this generic asset handler to be able to serve assets from different sources, like lambda, api etc...
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
}

func HandleAssets(req AssetsRequest) (AssetsResponse, error) {
	requestID := req.RequestID

	if !environments.ValidateEnvironment(req.Environment) {
		log.Printf("[RequestID: %s] Invalid environment: %s", requestID, req.Environment)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Invalid environment")}, nil
	}

	log.Printf("[RequestID: %s] Handling assets request for environment: %s", requestID, req.Environment)

	if req.AssetName == "" {
		log.Printf("[RequestID: %s] No asset name provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No asset name provided")}, nil
	}
	if req.Platform == "" || (req.Platform != "ios" && req.Platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, req.Platform)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Invalid platform")}, nil
	}
	if req.RuntimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No runtime version provided")}, nil
	}

	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion(req.Environment, req.RuntimeVersion)
	if err != nil || lastUpdate == nil {
		log.Printf("[RequestID: %s] No update found for runtimeVersion: %s", requestID, req.RuntimeVersion)
		return AssetsResponse{StatusCode: http.StatusNotFound, Body: []byte("No update found")}, nil
	}

	metadata, err := update.GetMetadata(*lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting metadata")}, nil
	}

	var platformMetadata types.PlatformMetadata
	switch req.Platform {
	case "android":
		platformMetadata = metadata.MetadataJSON.FileMetadata.Android
	case "ios":
		platformMetadata = metadata.MetadataJSON.FileMetadata.IOS
	default:
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Platform not supported")}, nil
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
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error resolving bucket")}, nil
	}

	asset, err := resolvedBucket.GetFile(*lastUpdate, req.AssetName)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting asset: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting asset")}, nil
	}

	buffer, err := bucket.ConvertReadCloserToBytes(asset.Reader)
	defer asset.Reader.Close()
	if err != nil {
		log.Printf("[RequestID: %s] Error converting asset to buffer: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error converting asset")}, nil
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
		Body:        buffer,
		ContentType: contentType,
	}, nil
}
