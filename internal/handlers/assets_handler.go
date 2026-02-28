package handlers

import (
	"expo-open-ota/internal/assets"
	"expo-open-ota/internal/helpers"
	cdn2 "expo-open-ota/internal/cdn"
	"expo-open-ota/internal/compression"
	"expo-open-ota/internal/channel"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
)

func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	channelName := r.Header.Get("expo-channel-name")
	if err := helpers.ValidateResourceName(channelName, "channel"); err != nil {
		log.Printf("[RequestID: %s] Invalid channel name: %v", requestID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	preventCDNRedirection := r.Header.Get("prevent-cdn-redirection") == "true"
	branchName, err := channel.GetChannelMapping(channelName)
	if err != nil {
		log.Printf("[RequestID: %s] No branch mapping found for channel %s: %v", requestID, channelName, err)
		http.Error(w, fmt.Sprintf("No branch mapping found for channel '%s'", channelName), http.StatusNotFound)
		return
	}

	req := assets.AssetsRequest{
		Branch:         branchName,
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
