package bucket

import (
	"cloud.google.com/go/storage"
	"context"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"fmt"
	"google.golang.org/api/iterator"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FirebaseBucket struct {
	BucketName string
}

func (b *FirebaseBucket) DeleteUpdateFolder(branch, runtimeVersion, updateId string) error {
	client, err := services.GetFirebaseStorageClient()
	if err != nil {
		return fmt.Errorf("error getting Firebase Storage client: %w", err)
	}
	ctx := context.Background()
	prefix := fmt.Sprintf("%s/%s/%s/", branch, runtimeVersion, updateId)
	it := client.Bucket(b.BucketName).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed listing objects: %w", err)
		}
		if err := client.Bucket(b.BucketName).Object(attrs.Name).Delete(ctx); err != nil {
			return fmt.Errorf("failed deleting object: %w", err)
		}
	}
	return nil
}

func (b *FirebaseBucket) GetBranches() ([]string, error) {
	client, err := services.GetFirebaseStorageClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	it := client.Bucket(b.BucketName).Objects(ctx, nil)
	branchesSet := make(map[string]struct{})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing branches: %w", err)
		}
		parts := strings.SplitN(attrs.Name, "/", 2)
		branch := parts[0]
		if branch != "" {
			branchesSet[branch] = struct{}{}
		}
	}
	branches := make([]string, 0, len(branchesSet))
	for branch := range branchesSet {
		branches = append(branches, branch)
	}
	sort.Strings(branches)
	return branches, nil
}

func (b *FirebaseBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	client, err := services.GetFirebaseStorageClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	prefixRoot := branch + "/"
	it := client.Bucket(b.BucketName).Objects(ctx, &storage.Query{Prefix: prefixRoot})
	versionsSet := make(map[string]struct{})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing runtime versions: %w", err)
		}
		name := attrs.Name[len(prefixRoot):]
		parts := strings.SplitN(name, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			versionsSet[parts[0]] = struct{}{}
		}
	}
	var runtimeVersions []RuntimeVersionWithStats
	for rv := range versionsSet {
		updatePrefix := prefixRoot + rv + "/"
		upIt := client.Bucket(b.BucketName).Objects(ctx, &storage.Query{Prefix: updatePrefix})
		var updateTimestamps []int64
		for {
			attrs, err := upIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("listing updates for %s: %w", rv, err)
			}
			name := attrs.Name[len(updatePrefix):]
			parts := strings.SplitN(name, "/", 2)
			idStr := parts[0]
			if ts, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				updateTimestamps = append(updateTimestamps, ts)
			}
		}
		if len(updateTimestamps) == 0 {
			continue
		}
		sort.Slice(updateTimestamps, func(i, j int) bool { return updateTimestamps[i] < updateTimestamps[j] })
		runtimeVersions = append(runtimeVersions, RuntimeVersionWithStats{
			RuntimeVersion:  rv,
			CreatedAt:       time.UnixMilli(updateTimestamps[0]).UTC().Format(time.RFC3339),
			LastUpdatedAt:   time.UnixMilli(updateTimestamps[len(updateTimestamps)-1]).UTC().Format(time.RFC3339),
			NumberOfUpdates: len(updateTimestamps),
		})
	}
	sort.Slice(runtimeVersions, func(i, j int) bool {
		return runtimeVersions[i].RuntimeVersion < runtimeVersions[j].RuntimeVersion
	})
	return runtimeVersions, nil
}

func (b *FirebaseBucket) GetUpdates(branch, runtimeVersion string) ([]types.Update, error) {
	client, err := services.GetFirebaseStorageClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	prefix := branch + "/" + runtimeVersion + "/"
	it := client.Bucket(b.BucketName).Objects(ctx, &storage.Query{Prefix: prefix})
	updatesSet := make(map[int64]struct{})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing updates: %w", err)
		}
		name := attrs.Name[len(prefix):]
		parts := strings.SplitN(name, "/", 2)
		idStr := parts[0]
		if ts, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			updatesSet[ts] = struct{}{}
		}
	}
	var timestamps []int64
	for ts := range updatesSet {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
	updates := make([]types.Update, len(timestamps))
	for i, ts := range timestamps {
		id := strconv.FormatInt(ts, 10)
		updates[i] = types.Update{
			Branch:         branch,
			RuntimeVersion: runtimeVersion,
			UpdateId:       id,
			CreatedAt:      time.Duration(ts) * time.Millisecond,
		}
	}
	return updates, nil
}

func (b *FirebaseBucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
	client, err := services.GetFirebaseStorageClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s/%s/%s/%s", update.Branch, update.RuntimeVersion, update.UpdateId, assetPath)
	rc, err := client.Bucket(b.BucketName).Object(key).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, nil
		}
		return nil, fmt.Errorf("GetObject error: %w", err)
	}
	attrs, err := client.Bucket(b.BucketName).Object(key).Attrs(ctx)
	if err != nil {
		rc.Close()
		return nil, fmt.Errorf("Attrs error: %w", err)
	}
	return &types.BucketFile{Reader: rc, CreatedAt: attrs.Updated}, nil
}
