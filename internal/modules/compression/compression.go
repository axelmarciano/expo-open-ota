package compression

import (
	"compress/gzip"
	"github.com/andybalholm/brotli"
	"log"
	"net/http"
	"strings"
)

func compressWithGzip(w http.ResponseWriter, data []byte, requestID string) error {
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()

	_, err := gz.Write(data)
	if err != nil {
		log.Printf("[RequestID: %s] Error compressing with Gzip: %v", requestID, err)
	}
	return err
}

func compressWithBrotli(w http.ResponseWriter, data []byte, requestID string) error {
	w.Header().Set("Content-Encoding", "br")
	br := brotli.NewWriter(w)
	defer br.Close()

	_, err := br.Write(data)
	if err != nil {
		log.Printf("[RequestID: %s] Error compressing with Brotli: %v", requestID, err)
	}
	return err
}

func ServeCompressedAsset(w http.ResponseWriter, r *http.Request, data []byte, contentType, requestID string) {
	w.Header().Set("Content-Type", contentType)
	acceptEncoding := r.Header.Get("Accept-Encoding")
	log.Printf("[RequestID: %s] Serving asset with content type: %s", requestID, contentType)

	if strings.Contains(acceptEncoding, "br") {
		if err := compressWithBrotli(w, data, requestID); err != nil {
			http.Error(w, "Error compressing with Brotli", http.StatusInternalServerError)
			return
		}
	} else if strings.Contains(acceptEncoding, "gzip") {
		if err := compressWithGzip(w, data, requestID); err != nil {
			http.Error(w, "Error compressing with Gzip", http.StatusInternalServerError)
			return
		}
	} else {
		_, err := w.Write(data)
		if err != nil {
			log.Printf("[RequestID: %s] Error writing uncompressed response: %v", requestID, err)
		}
	}
}
