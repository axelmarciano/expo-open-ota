package handlers

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/modules/assets"
	"expo-open-ota/internal/modules/certs"
	"expo-open-ota/internal/modules/compression"
	"expo-open-ota/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
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
}

func LambdaAssetsHandler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	assetRequest := assets.AssetsRequest{
		Environment:    req.PathParameters["environment"],
		AssetName:      req.QueryStringParameters["asset"],
		RuntimeVersion: req.QueryStringParameters["runtimeVersion"],
		Platform:       req.QueryStringParameters["platform"],
		RequestID:      req.RequestContext.RequestID,
	}
	cloudfrontDomain := config.GetEnv("CLOUDFRONT_DOMAIN")
	resp, err := assets.HandleAssetsWithURL(assetRequest, cloudfrontDomain)

	if err != nil {
		log.Printf("Error handling assets: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}

	headers := resp.Headers
	if resp.StatusCode != 200 {
		return events.APIGatewayProxyResponse{
			StatusCode: resp.StatusCode,
			Headers:    resp.Headers,
			Body:       `{"message": "` + string(resp.Body) + `"}`,
		}, nil
	}

	privateCloudfrontCert := certs.GetPrivateCloudfrontCert()
	if privateCloudfrontCert == "" {
		headers["Location"] = resp.URL
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusFound,
			Headers:    headers,
			Body:       "",
		}, nil
	}
	signedCookies, err := services.GenerateSignedCookies(resp.URL, privateCloudfrontCert)
	if err != nil {
		log.Printf("Error generating signed cookies: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}
	headers["Location"] = resp.URL
	headers["Set-Cookie"] = signedCookies
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound,
		Headers:    headers,
		Body:       "",
	}, nil
}
