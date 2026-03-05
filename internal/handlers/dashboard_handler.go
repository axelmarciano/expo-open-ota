package handlers

import (
	"encoding/json"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type BranchMapping struct {
	BranchName     string  `json:"branchName"`
	BranchId       *string `json:"branchId"`
	ReleaseChannel *string `json:"releaseChannel"`
}

type ChannelMapping struct {
	ReleaseChannelName string  `json:"releaseChannelName"`
	ReleaseChannelId   string  `json:"releaseChannelId"`
	BranchName         *string `json:"branchName"`
	BranchId           *string `json:"branchId"`
}

type UpdateItem struct {
	UpdateUUID string `json:"updateUUID"`
	UpdateId   string `json:"updateId"`
	CreatedAt  string `json:"createdAt"`
	CommitHash string `json:"commitHash"`
	Platform   string `json:"platform"`
	Message    string `json:"message,omitempty"`
}

type UpdateDetails struct {
	UpdateUUID string           `json:"updateUUID"`
	UpdateId   string           `json:"updateId"`
	CreatedAt  string           `json:"createdAt"`
	CommitHash string           `json:"commitHash"`
	Platform   string           `json:"platform"`
	Message    string           `json:"message,omitempty"`
	Type       types.UpdateType `json:"type"`
	ExpoConfig string           `json:"expoConfig"`
}

type SettingsEnv struct {
	BASE_URL                               string `json:"BASE_URL"`
	EXPO_APP_ID                            string `json:"EXPO_APP_ID"`
	EXPO_ACCESS_TOKEN                      string `json:"EXPO_ACCESS_TOKEN"`
	CACHE_MODE                             string `json:"CACHE_MODE"`
	REDIS_HOST                             string `json:"REDIS_HOST"`
	REDIS_PORT                             string `json:"REDIS_PORT"`
	STORAGE_MODE                           string `json:"STORAGE_MODE"`
	S3_BUCKET_NAME                         string `json:"S3_BUCKET_NAME"`
	LOCAL_BUCKET_BASE_PATH                 string `json:"LOCAL_BUCKET_BASE_PATH"`
	KEYS_STORAGE_TYPE                      string `json:"KEYS_STORAGE_TYPE"`
	AWSSM_EXPO_PUBLIC_KEY_SECRET_ID        string `json:"AWSSM_EXPO_PUBLIC_KEY_SECRET_ID"`
	AWSSM_EXPO_PRIVATE_KEY_SECRET_ID       string `json:"AWSSM_EXPO_PRIVATE_KEY_SECRET_ID"`
	PUBLIC_EXPO_KEY_B64                    string `json:"PUBLIC_EXPO_KEY_B64"`
	PUBLIC_LOCAL_EXPO_KEY_PATH             string `json:"PUBLIC_LOCAL_EXPO_KEY_PATH"`
	PRIVATE_LOCAL_EXPO_KEY_PATH            string `json:"PRIVATE_LOCAL_EXPO_KEY_PATH"`
	AWS_REGION                             string `json:"AWS_REGION"`
	AWS_BASE_ENDPOINT                      string `json:"AWS_BASE_ENDPOINT"`
	AWS_ACCESS_KEY_ID                      string `json:"AWS_ACCESS_KEY_ID"`
	CLOUDFRONT_DOMAIN                      string `json:"CLOUDFRONT_DOMAIN"`
	CLOUDFRONT_KEY_PAIR_ID                 string `json:"CLOUDFRONT_KEY_PAIR_ID"`
	CLOUDFRONT_PRIVATE_KEY_B64             string `json:"CLOUDFRONT_PRIVATE_KEY_B64"`
	AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID string `json:"AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"`
	PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH      string `json:"PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH"`
	PROMETHEUS_ENABLED                     string `json:"PROMETHEUS_ENABLED"`
}

func GetSettingsHandler(w http.ResponseWriter, r *http.Request) {

	// Retrieve all in config.GetEnv & return as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SettingsEnv{
		BASE_URL:    config.GetEnv("BASE_URL"),
		EXPO_APP_ID: config.GetEnv("EXPO_APP_ID"),
		// Only retrieve the first 5 characters of the token
		EXPO_ACCESS_TOKEN:                      "***" + config.GetEnv("EXPO_ACCESS_TOKEN")[:5],
		CACHE_MODE:                             config.GetEnv("CACHE_MODE"),
		REDIS_HOST:                             config.GetEnv("REDIS_HOST"),
		REDIS_PORT:                             config.GetEnv("REDIS_PORT"),
		STORAGE_MODE:                           config.GetEnv("STORAGE_MODE"),
		S3_BUCKET_NAME:                         config.GetEnv("S3_BUCKET_NAME"),
		LOCAL_BUCKET_BASE_PATH:                 config.GetEnv("LOCAL_BUCKET_BASE_PATH"),
		KEYS_STORAGE_TYPE:                      config.GetEnv("KEYS_STORAGE_TYPE"),
		AWSSM_EXPO_PUBLIC_KEY_SECRET_ID:        config.GetEnv("AWSSM_EXPO_PUBLIC_KEY_SECRET_ID"),
		AWSSM_EXPO_PRIVATE_KEY_SECRET_ID:       config.GetEnv("AWSSM_EXPO_PRIVATE_KEY_SECRET_ID"),
		PUBLIC_EXPO_KEY_B64:                    config.GetEnv("PUBLIC_EXPO_KEY_B64"),
		PUBLIC_LOCAL_EXPO_KEY_PATH:             config.GetEnv("PUBLIC_LOCAL_EXPO_KEY_PATH"),
		PRIVATE_LOCAL_EXPO_KEY_PATH:            config.GetEnv("PRIVATE_LOCAL_EXPO_KEY_PATH"),
		AWS_REGION:                             config.GetEnv("AWS_REGION"),
		AWS_BASE_ENDPOINT:                      config.GetEnv("AWS_BASE_ENDPOINT"),
		AWS_ACCESS_KEY_ID:                      config.GetEnv("AWS_ACCESS_KEY_ID"),
		CLOUDFRONT_DOMAIN:                      config.GetEnv("CLOUDFRONT_DOMAIN"),
		CLOUDFRONT_KEY_PAIR_ID:                 config.GetEnv("CLOUDFRONT_KEY_PAIR_ID"),
		CLOUDFRONT_PRIVATE_KEY_B64:             config.GetEnv("CLOUDFRONT_PRIVATE_KEY_B64"),
		AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID: config.GetEnv("AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"),
		PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH:      config.GetEnv("PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH"),
		PROMETHEUS_ENABLED:                     config.GetEnv("PROMETHEUS_ENABLED"),
	})
}

func GetChannelsHandler(w http.ResponseWriter, r *http.Request) {
	cacheKey := dashboard.ComputeGetChannelsCacheKey()
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var channels []ChannelMapping
		json.Unmarshal([]byte(cacheValue), &channels)
		json.NewEncoder(w).Encode(channels)
		return
	}
	allChannels, err := services.FetchExpoChannels()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	branchesMapping, err := services.FetchExpoBranchesMapping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var channels []ChannelMapping
	for _, channel := range allChannels {
		var branchName *string
		var branchId *string
		for _, mapping := range branchesMapping {
			if mapping.ChannelName != nil && *mapping.ChannelName == channel.Name {
				branchName = &mapping.BranchName
				branchId = &mapping.BranchId
				break
			}
		}
		channels = append(channels, ChannelMapping{
			ReleaseChannelId:   channel.Id,
			ReleaseChannelName: channel.Name,
			BranchName:         branchName,
			BranchId:           branchId,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(channels)
	ttl := 10 * time.Second
	ttlMs := int(ttl.Milliseconds())
	marshaledResponse, _ := json.Marshal(channels)
	cache.Set(cacheKey, string(marshaledResponse), &ttlMs)
}

func GetBranchesHandler(w http.ResponseWriter, r *http.Request) {
	resolvedBucket := bucket.GetBucket()
	branches, err := resolvedBucket.GetBranches()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	branchesMapping, err := services.FetchExpoBranchesMapping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var response []BranchMapping
	for _, branch := range branches {
		var releaseChannel *string
		var branchId *string
		for _, mapping := range branchesMapping {
			if mapping.BranchName == branch {
				releaseChannel = mapping.ChannelName
				branchId = &mapping.BranchId
				break
			}
		}
		response = append(response, BranchMapping{
			BranchName:     branch,
			BranchId:       branchId,
			ReleaseChannel: releaseChannel,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetRuntimeVersionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["BRANCH"]
	cacheKey := dashboard.ComputeGetRuntimeVersionsCacheKey(branchName)
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var runtimeVersions []bucket.RuntimeVersionWithStats
		json.Unmarshal([]byte(cacheValue), &runtimeVersions)
		json.NewEncoder(w).Encode(runtimeVersions)
		return
	}
	resolvedBucket := bucket.GetBucket()
	runtimeVersions, err := resolvedBucket.GetRuntimeVersions(branchName)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	sort.Slice(runtimeVersions, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, runtimeVersions[i].CreatedAt)
		timeJ, _ := time.Parse(time.RFC3339, runtimeVersions[j].CreatedAt)
		return timeI.After(timeJ)
	})
	json.NewEncoder(w).Encode(runtimeVersions)
	marshaledResponse, _ := json.Marshal(runtimeVersions)
	ttl := 10 * time.Second
	ttlMs := int(ttl.Milliseconds())
	cache.Set(cacheKey, string(marshaledResponse), &ttlMs)
}

func GetUpdateDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["BRANCH"]
	runtimeVersion := vars["RUNTIME_VERSION"]
	updateId := vars["UPDATE_ID"]
	cacheKey := dashboard.ComputeGetUpdateDetailsCacheKey(branchName, runtimeVersion, updateId)
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var updateDetailsResponse UpdateDetails
		json.Unmarshal([]byte(cacheValue), &updateDetailsResponse)
		json.NewEncoder(w).Encode(updateDetailsResponse)
		return
	}
	update, err := update2.GetUpdate(branchName, runtimeVersion, updateId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	metadata, err := update2.GetMetadata(*update)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	numberUpdate, _ := strconv.ParseInt(update.UpdateId, 10, 64)
	storedMetadata, _ := update2.RetrieveUpdateStoredMetadata(*update)
	expoConfig, err := update2.GetExpoConfig(*update)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updateUUID := storedMetadata.UpdateUUID
	if updateUUID == "" {
		updateUUID = crypto.ConvertSHA256HashToUUID(metadata.ID)
	}
	updatesResponse := UpdateDetails{
		UpdateUUID: updateUUID,
		UpdateId:   update.UpdateId,
		CreatedAt:  time.UnixMilli(numberUpdate).UTC().Format(time.RFC3339),
		CommitHash: storedMetadata.CommitHash,
		Platform:   storedMetadata.Platform,
		Message:    storedMetadata.Message,
		Type:       update2.GetUpdateType(*update),
		ExpoConfig: string(expoConfig),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatesResponse)
	marshaledResponse, _ := json.Marshal(updatesResponse)
	ttl := 120 * time.Second
	ttlMs := int(ttl.Milliseconds())
	cache.Set(cacheKey, string(marshaledResponse), &ttlMs)
}

func GetUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["BRANCH"]
	runtimeVersion := vars["RUNTIME_VERSION"]
	cacheKey := dashboard.ComputeGetUpdatesCacheKey(branchName, runtimeVersion)
	cache := cache2.GetCache()
	if cacheValue := cache.Get(cacheKey); cacheValue != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var updatesResponse []UpdateItem
		json.Unmarshal([]byte(cacheValue), &updatesResponse)
		json.NewEncoder(w).Encode(updatesResponse)
		return
	}
	resolvedBucket := bucket.GetBucket()
	updates, err := resolvedBucket.GetUpdates(branchName, runtimeVersion)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var updatesResponse []UpdateItem
	for _, update := range updates {
		isValid := update2.IsUpdateValid(update)
		if !isValid {
			continue
		}
		numberUpdate, _ := strconv.ParseInt(update.UpdateId, 10, 64)
		storedMetadata, _ := update2.RetrieveUpdateStoredMetadata(update)
		updateType := update2.GetUpdateType(update)
		if updateType == types.Rollback {
			updatesResponse = append(updatesResponse, UpdateItem{
				UpdateUUID: "Rollback to embedded",
				UpdateId:   update.UpdateId,
				CreatedAt:  time.UnixMilli(numberUpdate).UTC().Format(time.RFC3339),
				CommitHash: storedMetadata.CommitHash,
				Platform:   storedMetadata.Platform,
				Message:    storedMetadata.Message,
			})
			continue
		}

		metadata, err := update2.GetMetadata(update)
		if err != nil {
			continue
		}
		updateUUID := storedMetadata.UpdateUUID
		if updateUUID == "" {
			updateUUID = crypto.ConvertSHA256HashToUUID(metadata.ID)
		}
		updatesResponse = append(updatesResponse, UpdateItem{
			UpdateUUID: updateUUID,
			UpdateId:   update.UpdateId,
			CreatedAt:  time.UnixMilli(numberUpdate).UTC().Format(time.RFC3339),
			CommitHash: storedMetadata.CommitHash,
			Platform:   storedMetadata.Platform,
			Message:    storedMetadata.Message,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	sort.Slice(updatesResponse, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, updatesResponse[i].CreatedAt)
		timeJ, _ := time.Parse(time.RFC3339, updatesResponse[j].CreatedAt)
		return timeI.After(timeJ)
	})
	json.NewEncoder(w).Encode(updatesResponse)
	marshaledResponse, _ := json.Marshal(updatesResponse)
	ttl := 10 * time.Second
	ttlMs := int(ttl.Milliseconds())
	cache.Set(cacheKey, string(marshaledResponse), &ttlMs)
}

func UpdateChannelBranchMappingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchId := vars["BRANCH"]
	var requestBody struct {
		ReleaseChannel string `json:"releaseChannel"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		fmt.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error decoding request body"))
		return
	}
	releaseChannel := requestBody.ReleaseChannel
	if releaseChannel == "" {
		fmt.Println("Release channel is empty")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Release channel is empty"))
		return
	}
	err = services.UpdateChannelBranchMapping(releaseChannel, branchId)
	if err != nil {
		fmt.Println("Error updating channel branch mapping:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error updating channel branch mapping"))
		return
	}
	w.WriteHeader(http.StatusOK)
	marshaledResponse, _ := json.Marshal("ok")
	w.Header().Set("Content-Type", "application/json")
	w.Write(marshaledResponse)

	branchesCacheKey := dashboard.ComputeGetBranchesCacheKey()
	channelsCacheKey := dashboard.ComputeGetChannelsCacheKey()
	cache := cache2.GetCache()
	cache.Delete(branchesCacheKey)
	cache.Delete(channelsCacheKey)
}
