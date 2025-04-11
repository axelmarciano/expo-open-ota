package update

import (
	"encoding/json"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	cache2 "expo-open-ota/internal/cache"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/types"
	"fmt"
	"mime"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func sortUpdates(updates []types.Update) []types.Update {
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].CreatedAt > updates[j].CreatedAt
	})
	return updates
}

func GetAllUpdatesForRuntimeVersion(branch string, runtimeVersion string) ([]types.Update, error) {
	resolvedBucket := bucket.GetBucket()
	updates, errGetUpdates := resolvedBucket.GetUpdates(branch, runtimeVersion)
	if errGetUpdates != nil {
		return nil, errGetUpdates
	}
	updates = sortUpdates(updates)
	return updates, nil
}

func MarkUpdateAsChecked(update types.Update) error {
	cache := cache2.GetCache()
	branchesCacheKey := dashboard.ComputeGetBranchesCacheKey()
	runTimeVersionsCacheKey := dashboard.ComputeGetRuntimeVersionsCacheKey(update.Branch)
	updatesCacheKey := dashboard.ComputeGetUpdatesCacheKey(update.Branch, update.RuntimeVersion)
	cacheKeys := []string{ComputeLastUpdateCacheKey(update.Branch, update.RuntimeVersion), branchesCacheKey, runTimeVersionsCacheKey, updatesCacheKey}
	for _, cacheKey := range cacheKeys {
		cache.Delete(cacheKey)
	}
	resolvedBucket := bucket.GetBucket()
	reader := strings.NewReader(".check")
	_ = resolvedBucket.UploadFileIntoUpdate(update, ".check", reader)
	return nil
}

func IsUpdateValid(Update types.Update) bool {
	resolvedBucket := bucket.GetBucket()
	// Search for .check file in the update
	file, _ := resolvedBucket.GetFile(Update, ".check")
	if file.Reader != nil {
		defer file.Reader.Close()
		return true
	}
	return false
}

func ComputeLastUpdateCacheKey(branch string, runtimeVersion string) string {
	return fmt.Sprintf("lastUpdate:%s:%s", branch, runtimeVersion)
}

func ComputeMetadataCacheKey(branch string, runtimeVersion string, updateId string) string {
	return fmt.Sprintf("metadata:%s:%s:%s", branch, runtimeVersion, updateId)
}

func ComputeUpdataManifestCacheKey(branch string, runtimeVersion string, updateId string, platform string) string {
	return fmt.Sprintf("manifest:%s:%s:%s:%s", branch, runtimeVersion, updateId, platform)
}

func ComputeManifestAssetCacheKey(update types.Update, assetPath string) string {
	return fmt.Sprintf("asset:%s:%s:%s:%s", update.Branch, update.RuntimeVersion, update.UpdateId, assetPath)
}

func VerifyUploadedUpdate(update types.Update) error {
	metadata, errMetadata := GetMetadata(update)
	if errMetadata != nil {
		return errMetadata
	}
	if metadata.MetadataJSON.FileMetadata.IOS.Bundle == "" && metadata.MetadataJSON.FileMetadata.Android.Bundle == "" {
		return fmt.Errorf("missing bundle path in metadata")
	}
	files := []string{}
	if metadata.MetadataJSON.FileMetadata.IOS.Bundle != "" {
		files = append(files, metadata.MetadataJSON.FileMetadata.IOS.Bundle)
		for _, asset := range metadata.MetadataJSON.FileMetadata.IOS.Assets {
			files = append(files, asset.Path)
		}
	}
	if metadata.MetadataJSON.FileMetadata.Android.Bundle != "" {
		files = append(files, metadata.MetadataJSON.FileMetadata.Android.Bundle)
		for _, asset := range metadata.MetadataJSON.FileMetadata.Android.Assets {
			files = append(files, asset.Path)
		}
	}

	resolvedBucket := bucket.GetBucket()
	for _, file := range files {
		_, err := resolvedBucket.GetFile(update, file)
		if err != nil {
			return fmt.Errorf("missing file: %s in update", file)
		}
	}
	return nil
}

func GetUpdate(branch string, runtimeVersion string, updateId string) (*types.Update, error) {
	updateIdInt64, err := strconv.ParseInt(updateId, 10, 64)
	if err != nil {
		return nil, err
	}
	return &types.Update{
		Branch:         branch,
		RuntimeVersion: runtimeVersion,
		UpdateId:       updateId,
		CreatedAt:      time.Duration(updateIdInt64) * time.Millisecond,
	}, nil
}

func AreUpdatesIdentical(update1, update2 types.Update, platform string) (bool, error) {
	metadata1, errMetadata1 := GetMetadata(update1)
	if errMetadata1 != nil {
		return false, errMetadata1
	}
	metadata2, errMetadata2 := GetMetadata(update2)
	if errMetadata2 != nil {
		return false, errMetadata2
	}
	update1Manifest, errManifest1 := ComposeUpdateManifest(&metadata1, update1, platform)
	if errManifest1 != nil {
		return false, errManifest1
	}
	update2Manifest, errManifest2 := ComposeUpdateManifest(&metadata2, update2, platform)
	if errManifest2 != nil {
		return false, errManifest2
	}
	if update1Manifest.LaunchAsset.Hash != update2Manifest.LaunchAsset.Hash {
		return false, nil
	}
	if len(update2Manifest.Assets) != len(update1Manifest.Assets) {
		return false, nil
	}
	for i, asset := range update1Manifest.Assets {
		if asset.Hash != update2Manifest.Assets[i].Hash {
			return false, nil
		}
	}
	return true, nil
}

func GetLatestUpdateBundlePathForRuntimeVersion(branch string, runtimeVersion string) (*types.Update, error) {
	cache := cache2.GetCache()
	cacheKey := fmt.Sprintf(ComputeLastUpdateCacheKey(branch, runtimeVersion))
	if cachedValue := cache.Get(cacheKey); cachedValue != "" {
		var update types.Update
		err := json.Unmarshal([]byte(cachedValue), &update)
		if err != nil {
			return nil, err
		}
		return &update, nil
	}
	updates, err := GetAllUpdatesForRuntimeVersion(branch, runtimeVersion)
	if err != nil {
		return nil, err
	}
	filteredUpdates := make([]types.Update, 0)
	for _, update := range updates {
		if IsUpdateValid(update) {
			filteredUpdates = append(filteredUpdates, update)
		}
	}
	if len(filteredUpdates) > 0 {
		cacheValue, err := json.Marshal(filteredUpdates[0])
		if err != nil {
			return &filteredUpdates[0], nil
		}
		ttl := 1800
		err = cache.Set(cacheKey, string(cacheValue), &ttl)
		return &filteredUpdates[0], nil
	}
	return nil, nil
}

func GetUpdateType(update types.Update) types.UpdateType {
	resolvedBucket := bucket.GetBucket()
	file, err := resolvedBucket.GetFile(update, "rollback")
	if err == nil && file.Reader != nil {
		defer file.Reader.Close()
		return types.Rollback
	}
	return types.NormalUpdate
}

func GetExpoConfig(update types.Update) (json.RawMessage, error) {
	resolvedBucket := bucket.GetBucket()
	resp, err := resolvedBucket.GetFile(update, "expoConfig.json")
	if err != nil {
		return nil, err
	}
	defer resp.Reader.Close()
	var expoConfig json.RawMessage
	err = json.NewDecoder(resp.Reader).Decode(&expoConfig)
	if err != nil {
		return nil, err
	}
	return expoConfig, nil
}

func GetMetadata(update types.Update) (types.UpdateMetadata, error) {
	metadataCacheKey := ComputeMetadataCacheKey(update.Branch, update.RuntimeVersion, update.UpdateId)
	cache := cache2.GetCache()
	if cachedValue := cache.Get(metadataCacheKey); cachedValue != "" {
		var metadata types.UpdateMetadata
		err := json.Unmarshal([]byte(cachedValue), &metadata)
		if err != nil {
			return types.UpdateMetadata{}, err
		}
		return metadata, nil
	}
	resolvedBucket := bucket.GetBucket()
	file, errFile := resolvedBucket.GetFile(update, "metadata.json")
	if errFile != nil {
		return types.UpdateMetadata{}, errFile
	}
	createdAt := file.CreatedAt
	var metadata types.UpdateMetadata
	var metadataJson types.MetadataObject
	err := json.NewDecoder(file.Reader).Decode(&metadataJson)
	defer file.Reader.Close()
	if err != nil {
		fmt.Println("error decoding metadata json:", err)
		return types.UpdateMetadata{}, err
	}
	metadata.CreatedAt = createdAt.UTC().Format("2006-01-02T15:04:05.000Z")
	metadata.MetadataJSON = metadataJson
	stringifiedMetadata, err := json.Marshal(metadata.MetadataJSON)
	if err != nil {
		return types.UpdateMetadata{}, err
	}
	id, errHash := crypto.CreateHash(stringifiedMetadata, "sha256", "hex")

	if errHash != nil {
		return types.UpdateMetadata{}, errHash
	}
	metadata.ID = id
	cacheValue, err := json.Marshal(metadata)
	if err != nil {
		return metadata, nil
	}
	err = cache.Set(metadataCacheKey, string(cacheValue), nil)
	return metadata, nil
}

func BuildFinalManifestAssetUrlURL(baseURL, assetFilePath, runtimeVersion, platform string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	query := url.Values{}
	query.Set("asset", assetFilePath)
	query.Set("runtimeVersion", runtimeVersion)
	query.Set("platform", platform)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func GetAssetEndpoint() string {
	return config.GetEnv("BASE_URL") + "/assets"
}

func shapeManifestAsset(update types.Update, asset *types.Asset, isLaunchAsset bool, platform string) (types.ManifestAsset, error) {
	cacheKey := ComputeManifestAssetCacheKey(update, asset.Path)
	cache := cache2.GetCache()
	if cachedValue := cache.Get(cacheKey); cachedValue != "" {
		var manifestAsset types.ManifestAsset
		err := json.Unmarshal([]byte(cachedValue), &manifestAsset)
		if err != nil {
			return types.ManifestAsset{}, err
		}
		return manifestAsset, nil
	}
	resolvedBucket := bucket.GetBucket()
	assetFilePath := asset.Path
	assetFile, errAssetFile := resolvedBucket.GetFile(update, asset.Path)
	if errAssetFile != nil {
		return types.ManifestAsset{}, errAssetFile
	}

	byteAsset, errAsset := bucket.ConvertReadCloserToBytes(assetFile.Reader)
	defer assetFile.Reader.Close()
	if errAsset != nil {
		return types.ManifestAsset{}, errAsset
	}
	assetHash, errHash := crypto.CreateHash(byteAsset, "sha256", "base64")
	if errHash != nil {
		return types.ManifestAsset{}, errHash
	}
	urlEncodedHash := crypto.GetBase64URLEncoding(assetHash)
	key, errKey := crypto.CreateHash(byteAsset, "md5", "hex")
	if errKey != nil {
		return types.ManifestAsset{}, errKey
	}

	keyExtensionSuffix := asset.Ext
	if isLaunchAsset {
		keyExtensionSuffix = "bundle"
	}
	keyExtensionSuffix = "." + keyExtensionSuffix
	contentType := "application/javascript"
	if isLaunchAsset {
		contentType = mime.TypeByExtension(asset.Ext)
	}
	finalUrl, errUrl := BuildFinalManifestAssetUrlURL(GetAssetEndpoint(), assetFilePath, update.RuntimeVersion, platform)
	if errUrl != nil {
		return types.ManifestAsset{}, errUrl
	}
	manifestAsset := types.ManifestAsset{
		Hash:          urlEncodedHash,
		Key:           key,
		FileExtension: keyExtensionSuffix,
		ContentType:   contentType,
		Url:           finalUrl,
	}
	cacheValue, err := json.Marshal(manifestAsset)
	if err != nil {
		return manifestAsset, nil
	}
	_ = cache.Set(cacheKey, string(cacheValue), nil)
	return manifestAsset, nil
}

func appendChannelOverrideToUrl(urlStr, channelOverride string) string {
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	query := parsedUrl.Query()
	query.Set("ow-expo-channel", channelOverride)
	parsedUrl.RawQuery = query.Encode()
	return parsedUrl.String()
}

func AppendChannelOverrideToAsset(manifest *types.UpdateManifest, channelOverride string) {
	manifest.LaunchAsset.Url = appendChannelOverrideToUrl(manifest.LaunchAsset.Url, channelOverride)
	for i := range manifest.Assets {
		manifest.Assets[i].Url = appendChannelOverrideToUrl(manifest.Assets[i].Url, channelOverride)
	}
}

func ComposeUpdateManifest(
	metadata *types.UpdateMetadata,
	update types.Update,
	platform string,
) (types.UpdateManifest, error) {
	cache := cache2.GetCache()
	cacheKey := ComputeUpdataManifestCacheKey(update.Branch, update.RuntimeVersion, update.UpdateId, platform)
	if cachedValue := cache.Get(cacheKey); cachedValue != "" {
		var manifest types.UpdateManifest
		err := json.Unmarshal([]byte(cachedValue), &manifest)
		if err != nil {
			return types.UpdateManifest{}, err
		}
		return manifest, nil
	}
	expoConfig, errConfig := GetExpoConfig(update)
	if errConfig != nil {
		return types.UpdateManifest{}, errConfig
	}

	var platformSpecificMetadata types.PlatformMetadata
	switch platform {
	case "ios":
		platformSpecificMetadata = metadata.MetadataJSON.FileMetadata.IOS
	case "android":
		platformSpecificMetadata = metadata.MetadataJSON.FileMetadata.Android
	}
	var (
		assets = make([]types.ManifestAsset, len(platformSpecificMetadata.Assets))
		errs   = make(chan error, len(platformSpecificMetadata.Assets))
		wg     sync.WaitGroup
	)

	for i, a := range platformSpecificMetadata.Assets {
		wg.Add(1)
		go func(index int, asset types.Asset) {
			defer wg.Done()
			shapedAsset, errShape := shapeManifestAsset(update, &asset, false, platform)
			if errShape != nil {
				errs <- errShape
				return
			}
			assets[index] = shapedAsset
		}(i, a)
	}

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		return types.UpdateManifest{}, <-errs
	}

	launchAsset, errShape := shapeManifestAsset(update, &types.Asset{
		Path: platformSpecificMetadata.Bundle,
		Ext:  "",
	}, true, platform)
	if errShape != nil {
		return types.UpdateManifest{}, errShape
	}

	manifest := types.UpdateManifest{
		Id:             crypto.ConvertSHA256HashToUUID(metadata.ID),
		CreatedAt:      metadata.CreatedAt,
		RunTimeVersion: update.RuntimeVersion,
		Metadata:       json.RawMessage("{}"),
		Extra: types.ExtraManifestData{
			ExpoClient: expoConfig,
			Branch:     update.Branch,
		},
		Assets:      assets,
		LaunchAsset: launchAsset,
	}
	cacheValue, err := json.Marshal(manifest)
	if err != nil {
		return manifest, nil
	}
	_ = cache.Set(cacheKey, string(cacheValue), nil)

	return manifest, nil
}

func CreateRollbackDirective(update types.Update) (types.RollbackDirective, error) {
	resolvedBucket := bucket.GetBucket()
	object, err := resolvedBucket.GetFile(update, "rollback")
	if err != nil {
		return types.RollbackDirective{}, err
	}
	commitTime := object.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z")
	defer object.Reader.Close()
	return types.RollbackDirective{
		Type: "rollBackToEmbedded",
		Parameters: types.RollbackDirectiveParameters{
			CommitTime: commitTime,
		},
	}, nil
}

func CreateNoUpdateAvailableDirective() types.NoUpdateAvailableDirective {
	return types.NoUpdateAvailableDirective{
		Type: "noUpdateAvailable",
	}
}

func RetrieveUpdateCommitHashAndPlatform(update types.Update) (string, string, error) {
	resolvedBucket := bucket.GetBucket()
	file, err := resolvedBucket.GetFile(update, "update-metadata.json")
	if err != nil {
		return "", "", err
	}
	defer file.Reader.Close()
	var metadata struct {
		Platform   string `json:"platform"`
		CommitHash string `json:"commitHash"`
	}
	err = json.NewDecoder(file.Reader).Decode(&metadata)
	if err != nil {
		return "", "", err
	}
	return metadata.CommitHash, metadata.Platform, nil
}
