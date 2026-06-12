package services

import (
	"context"
	update2 "expo-open-ota/internal/update"
	"log"
)

// PreWarmManifestCache populates the manifest cache layers for the given
// appId/branch/runtimeVersion/platform combination. It is intended to be
// called as a goroutine after MarkUpdateAsChecked so the first client
// request hits warm caches instead of rebuilding everything from scratch.
func PreWarmManifestCache(updateService *UpdateService, appId string, branch string, runtimeVersion string, platform string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PreWarm] panic recovered for app=%s branch=%s rv=%s platform=%s: %v", appId, branch, runtimeVersion, platform, r)
		}
	}()

	ctx := context.Background()
	latestUpdate, err := updateService.GetLatestUpdate(ctx, appId, branch, runtimeVersion, platform)
	if err != nil {
		log.Printf("[PreWarm] error getting latest update for app=%s branch=%s rv=%s platform=%s: %v", appId, branch, runtimeVersion, platform, err)
		return
	}
	if latestUpdate == nil {
		return
	}

	metadata, err := update2.GetMetadata(*latestUpdate)
	if err != nil {
		log.Printf("[PreWarm] error getting metadata for update=%s: %v", latestUpdate.UpdateId, err)
		return
	}

	_, err = update2.ComposeUpdateManifest(&metadata, *latestUpdate, platform)
	if err != nil {
		log.Printf("[PreWarm] error composing manifest for update=%s platform=%s: %v", latestUpdate.UpdateId, platform, err)
		return
	}

	log.Printf("[PreWarm] successfully pre-warmed cache for branch=%s rv=%s platform=%s", branch, runtimeVersion, platform)
}
