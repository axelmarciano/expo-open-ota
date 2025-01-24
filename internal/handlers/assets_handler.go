package handlers

import (
	"expo-open-ota/internal/modules/assets"
	"expo-open-ota/internal/modules/compression"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
)

func AssetsHandler(w http.ResponseWriter, r *http.Request) {
	req := assets.AssetsRequest{
		Environment:    mux.Vars(r)["ENVIRONMENT"],
		AssetName:      r.URL.Query().Get("asset"),
		RuntimeVersion: r.URL.Query().Get("runtimeVersion"),
		Platform:       r.URL.Query().Get("platform"),
		RequestID:      uuid.New().String(),
	}

	resp, err := assets.HandleAssets(req)
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
}
