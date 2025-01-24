package handlers

import (
	"expo-open-ota/internal/modules/bucket"
	"expo-open-ota/internal/modules/compression"
	"expo-open-ota/internal/modules/environments"
	"expo-open-ota/internal/modules/types"
	"expo-open-ota/internal/modules/update"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"mime"
	"net/http"
)

func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	vars := mux.Vars(r)
	environment := vars["ENVIRONMENT"]
	if !environments.ValidateEnvironment(environment) {
		log.Printf("[RequestID: %s] Invalid environment: %s", requestID, environment)
		http.Error(w, "Invalid environment", http.StatusBadRequest)
		return
	}
	log.Printf("[RequestID: %s] Handling assets request for environment: %s", requestID, environment)

	assetName := r.URL.Query().Get("asset")
	runtimeVersion := r.URL.Query().Get("runtimeVersion")
	platform := r.URL.Query().Get("platform")

	if assetName == "" {
		log.Printf("[RequestID: %s] No asset name provided", requestID)
		http.Error(w, "No asset name provided", http.StatusBadRequest)
		return
	}
	if platform == "" || (platform != "ios" && platform != "android") {
		log.Printf("[RequestID: %s] Invalid platform: %s", requestID, platform)
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}
	if runtimeVersion == "" {
		log.Printf("[RequestID: %s] No runtime version provided", requestID)
		http.Error(w, "No runtime version provided", http.StatusBadRequest)
		return
	}

	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion(environment, runtimeVersion)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting latest update: %v", requestID, err)
		http.Error(w, "Error getting latest update", http.StatusInternalServerError)
		return
	}
	if lastUpdate == nil {
		log.Printf("[RequestID: %s] No update found for runtimeVersion: %s", requestID, runtimeVersion)
		http.Error(w, "No update found", http.StatusNotFound)
		return
	}

	metadata, err := update.GetMetadata(*lastUpdate)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting metadata: %v", requestID, err)
		http.Error(w, "Error getting metadata", http.StatusInternalServerError)
		return
	}

	var platformMetadata types.PlatformMetadata
	switch platform {
	case "android":
		platformMetadata = metadata.MetadataJSON.FileMetadata.Android
	case "ios":
		platformMetadata = metadata.MetadataJSON.FileMetadata.IOS
	default:
		log.Printf("[RequestID: %s] Unsupported platform: %s", requestID, platform)
		http.Error(w, "Platform not supported", http.StatusBadRequest)
		return
	}

	bundle := platformMetadata.Bundle

	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		log.Printf("[RequestID: %s] Error resolving bucket: %v", requestID, errResolveBucket)
		http.Error(w, "Error resolving bucket", http.StatusInternalServerError)
		return
	}

	isLaunchAsset := bundle == assetName

	var assetMetadata types.Asset
	for _, asset := range platformMetadata.Assets {
		if asset.Path == assetName {
			assetMetadata = asset
		}
	}

	asset, err := resolvedBucket.GetFile(*lastUpdate, assetName)
	if err != nil {
		log.Printf("[RequestID: %s] Error getting asset: %v", requestID, err)
		http.Error(w, "Error getting asset", http.StatusInternalServerError)
		return
	}
	buffer, err := bucket.ConvertReadCloserToBytes(asset.Reader)
	defer asset.Reader.Close()
	if err != nil {
		log.Printf("[RequestID: %s] Error converting asset to buffer: %v", requestID, err)
		http.Error(w, "Error converting asset to buffer", http.StatusInternalServerError)
		return
	}

	var contentType string

	if isLaunchAsset {
		contentType = "application/javascript"
	} else {
		contentType = mime.TypeByExtension("." + string(assetMetadata.Ext))
	}
	w.Header().Set("expo-protocol-version", "1")
	w.Header().Set("expo-sfv-version", "0")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	log.Printf("[RequestID: %s] Serving asset: %s", requestID, assetName)
	compression.ServeCompressedAsset(w, r, buffer, contentType, requestID)
}
