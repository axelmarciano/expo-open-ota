package handlers

import (
	"expo-open-ota/internal/assets"
	cdn2 "expo-open-ota/internal/cdn"
	"expo-open-ota/internal/compression"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
	"github.com/google/uuid"
	"log"
	"net/http"
)

func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	channelName := helpers.ResolveExpoChannel(r.Header)
	expoChannelOverride := r.URL.Query().Get("ow-expo-channel")
	if expoChannelOverride != "" {
		channelName = expoChannelOverride
	}
	preventCDNRedirection := r.Header.Get("prevent-cdn-redirection") == "true"
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

	req := assets.AssetsRequest{
		Branch:         branchMap.BranchName,
		AssetName:      r.URL.Query().Get("asset"),
		RuntimeVersion: r.URL.Query().Get("runtimeVersion"),
		Platform:       r.URL.Query().Get("platform"),
		RequestID:      requestID,
	}

	cdn := cdn2.GetCDN()
	if cdn == nil || preventCDNRedirection {
		resp, err := assets.HandleAssetsWithFile(req)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		for key, value := range resp.Headers {
			w.Header().Set(key, value)
		}
		if resp.StatusCode != 200 {
			http.Error(w, string(resp.Body), resp.StatusCode)
			return
		}
		compression.ServeCompressedAsset(w, r, resp.Body, resp.ContentType, req.RequestID)
		return
	}
	resp, err := assets.HandleAssetsWithURL(req, cdn)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, resp.URL, http.StatusFound)
}
