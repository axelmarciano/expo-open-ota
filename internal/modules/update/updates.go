package update

import (
	"encoding/json"
	"expo-open-ota/config"
	"expo-open-ota/internal/modules/bucket"
	"expo-open-ota/internal/modules/crypto"
	"expo-open-ota/internal/modules/types"
	"fmt"
	"mime"
	"net/url"
	"sort"
	"sync"
)

func sortUpdates(updates []types.Update) []types.Update {
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].CreatedAt > updates[j].CreatedAt
	})
	return updates
}

func GetAllUpdatesForRuntimeVersion(environment string, runtimeVersion string) ([]types.Update, error) {
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return nil, errResolveBucket
	}
	updates, errGetUpdates := resolvedBucket.GetUpdates(environment, runtimeVersion)
	if errGetUpdates != nil {
		return nil, errGetUpdates
	}
	updates = sortUpdates(updates)
	return updates, nil
}

func GetLatestUpdateBundlePathForRuntimeVersion(environment string, runtimeVersion string) (*types.Update, error) {
	updates, err := GetAllUpdatesForRuntimeVersion(environment, runtimeVersion)
	if err != nil {
		return nil, err
	}
	if len(updates) > 0 {
		return &updates[0], nil
	}
	return nil, nil
}

func GetUpdateType(update types.Update) types.UpdateType {
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return types.NormalUpdate
	}
	file, err := resolvedBucket.GetFile(update, "rollback")
	if err == nil && file.Reader != nil {
		defer file.Reader.Close()
		return types.Rollback
	}
	return types.NormalUpdate
}

func GetExpoConfig(update types.Update) (json.RawMessage, error) {
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return nil, errResolveBucket
	}
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
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return types.UpdateMetadata{}, errResolveBucket
	}
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
	metadata.CreatedAt = createdAt.Format("2006-01-02T15:04:05.000Z")
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
	return metadata, nil
}

func BuildFinalManifestAssetUrlURL(baseURL, environment, assetFilePath, runtimeVersion, platform string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL.Path, err = url.JoinPath(parsedURL.Path, "assets", environment)
	if err != nil {
		return "", fmt.Errorf("error joining path: %w", err)
	}

	query := url.Values{}
	query.Set("asset", assetFilePath)
	query.Set("runtimeVersion", runtimeVersion)
	query.Set("platform", platform)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

func shapeManifestAsset(update types.Update, asset *types.Asset, isLaunchAsset bool, platform string) (types.ManifestAsset, error) {
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return types.ManifestAsset{}, errResolveBucket
	}
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
	finalUrl, errUrl := BuildFinalManifestAssetUrlURL(config.GetEnv("BASE_URL"), update.Environment, assetFilePath, update.RuntimeVersion, platform)
	if errUrl != nil {
		return types.ManifestAsset{}, errUrl
	}
	return types.ManifestAsset{
		Hash:          urlEncodedHash,
		Key:           key,
		FileExtension: keyExtensionSuffix,
		ContentType:   contentType,
		Url:           finalUrl,
	}, nil
}

func ComposeUpdateManifest(
	metadata *types.UpdateMetadata,
	update types.Update,
	platform string,
) (types.UpdateManifest, error) {
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
		},
		Assets:      assets,
		LaunchAsset: launchAsset,
	}
	return manifest, nil
}

func CreateRollbackDirective(update types.Update) (types.RollbackDirective, error) {
	resolvedBucket, errResolveBucket := bucket.GetBucket()
	if errResolveBucket != nil {
		return types.RollbackDirective{}, errResolveBucket
	}
	object, err := resolvedBucket.GetFile(update, "rollback")
	if err != nil {
		return types.RollbackDirective{}, err
	}
	commitTime := object.CreatedAt.Format("2006-01-02T15:04:05.000Z")
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
