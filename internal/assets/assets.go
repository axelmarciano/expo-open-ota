package assets

import (
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/cdn"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/update"
	"log"
	"mime"
	"net/http"
)

type AssetsRequest struct {
	AppId          string
	Branch         string
	AssetName      string
	RuntimeVersion string
	Platform       string
	RequestID      string
	Update         *types.Update
}

type AssetsResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        []byte
	ContentType string
	URL         string
}

func getAssetMetadata(req AssetsRequest, returnAsset bool) (AssetsResponse, *types.BucketFile, string, error) {
	requestID := req.RequestID

	if req.AssetName == "" {
		log.Printf("[RequestID: %s] No asset name provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No asset name provided")}, nil, "", nil
	}

	if req.Platform == "" || (req.Platform != "ios" && req.Platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, req.Platform)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Invalid platform")}, nil, "", nil
	}

	if req.RuntimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("No runtime version provided")}, nil, "", nil
	}

	lastUpdate := req.Update
	if lastUpdate == nil {
		// The service resolves the update and 404s before reaching here, so
		// a nil Update means "no matching update". Guard it rather than
		// dereferencing into a SIGSEGV below.
		log.Printf("[RequestID: %s] No update found", requestID)
		return AssetsResponse{StatusCode: http.StatusNotFound, Body: []byte("No update found")}, nil, "", nil
	}
	if !returnAsset {
		headers := map[string]string{
			"expo-protocol-version": "1",
			"expo-sfv-version":      "0",
			"Cache-Control":         "public, max-age=31536000",
		}
		return AssetsResponse{
			StatusCode: http.StatusOK,
			Headers:    headers,
		}, nil, lastUpdate.UpdateId, nil
	}

	metadata, err := update.GetMetadata(*lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting metadata")}, nil, "", nil
	}

	var platformMetadata types.PlatformMetadata
	switch req.Platform {
	case "android":
		platformMetadata = metadata.MetadataJSON.FileMetadata.Android
	case "ios":
		platformMetadata = metadata.MetadataJSON.FileMetadata.IOS
	default:
		return AssetsResponse{StatusCode: http.StatusBadRequest, Body: []byte("Platform not supported")}, nil, "", nil
	}

	bundle := platformMetadata.Bundle
	isLaunchAsset := bundle == req.AssetName

	var assetMetadata types.Asset
	for _, asset := range platformMetadata.Assets {
		if asset.Path == req.AssetName {
			assetMetadata = asset
		}
	}

	if !returnAsset {
		if !isLaunchAsset && assetMetadata == (types.Asset{}) {
			log.Printf("[RequestID: %s] Asset not found in metadata: %s", requestID, req.AssetName)
			return AssetsResponse{StatusCode: http.StatusNotFound, Body: []byte("Asset not found")}, nil, "", nil
		}
		headers := map[string]string{
			"expo-protocol-version": "1",
			"expo-sfv-version":      "0",
			"Cache-Control":         "public, max-age=31536000",
		}
		return AssetsResponse{
			StatusCode: http.StatusOK,
			Headers:    headers,
		}, nil, lastUpdate.UpdateId, nil
	}

	resolvedBucket := bucket.GetBucket()
	asset, err := resolvedBucket.GetFile(*lastUpdate, req.AssetName)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting asset: %v", requestID, err)
		return AssetsResponse{StatusCode: http.StatusInternalServerError, Body: []byte("Error getting asset")}, nil, "", nil
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
	}, asset, lastUpdate.UpdateId, nil
}

func HandleAssetsWithFile(req AssetsRequest) (AssetsResponse, error) {
	resp, asset, _, err := getAssetMetadata(req, true)
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

func HandleAssetsWithURL(req AssetsRequest, resolvedCDN cdn.CDN) (AssetsResponse, error) {
	resp, _, updateId, err := getAssetMetadata(req, false)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != 200 {
		return AssetsResponse{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
		}, nil
	}
	resp.URL, err = resolvedCDN.ComputeRedirectionURLForAsset(req.AppId, req.Branch, req.RuntimeVersion, updateId, req.AssetName)
	if err != nil {
		log.Printf("[RequestID: %s] Error computing redirection URL: %v", req.RequestID, err)
		return AssetsResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       []byte("Error computing redirection URL"),
		}, err
	}
	return resp, nil
}
