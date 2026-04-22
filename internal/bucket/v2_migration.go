package bucket

import (
	"context"
	"expo-open-ota/internal/services"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/api/iterator"
)

// v2_migration.go — one-shot data re-path from the v1 bucket layout
// ({prefix}/{branch}/{rv}/{updateId}/…) to the v2 layout
// ({prefix}/{appId}/{branch}/{rv}/{updateId}/…). Driven by the migration
// 20260422_v2_scope_data_under_appid registered in internal/migrations/.
//
// Each backend exposes a MoveRootEntriesUnder(appId) method that is
// idempotent under interruption: entries already moved are detected
// (the appId prefix is preserved on retry) and re-running the migration
// only processes what's still at the root.
//
// .migrationhistory itself is deployment-global and explicitly excluded —
// it must stay at the bucket root so the migration ledger keeps working
// across deploys.

// MoveRootEntriesUnder walks the LocalBucket root and moves every
// immediate child directory that LOOKS LIKE a v1 branch into
// {rootPath}/{appId}/. An entry is considered a v1 branch when it
// contains a .check or update-metadata.json file at depth 3 — the exact
// shape produced by the v1 publish pipeline
// ({branch}/{runtimeVersion}/{updateId}/.check). v2 directories hold the
// same files at depth 4 ({appId}/{branch}/{rv}/{updateId}/.check) and so
// are correctly identified as non-branch-shaped and left alone. Uses
// os.Rename per entry, atomic on POSIX when src/dst share a filesystem.
func (b *LocalBucket) MoveRootEntriesUnder(appId string) error {
	root := b.rootPath()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", root, err)
	}

	// Figure out what actually needs moving before creating the target
	// dir — on a bucket that's already fully v2 we don't want to litter
	// an empty {appId}/ entry.
	var toMove []string
	for _, e := range entries {
		name := e.Name()
		if name == appId || name == ".migrationhistory" || !e.IsDir() {
			continue
		}
		if b.looksLikeV1Branch(name) {
			toMove = append(toMove, name)
		}
	}
	if len(toMove) == 0 {
		return nil
	}

	targetDir := filepath.Join(root, appId)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir %s: %w", targetDir, err)
	}
	for _, name := range toMove {
		src := filepath.Join(root, name)
		dst := filepath.Join(targetDir, name)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %s: %w", name, err)
		}
	}
	return nil
}

// looksLikeV1Branch returns true when {rootPath}/{name} contains a .check
// or update-metadata.json file at depth 3 (branch/rv/updateId/.check).
// The check short-circuits on the first match so a branch with many
// runtime versions is cheap to classify.
func (b *LocalBucket) looksLikeV1Branch(name string) bool {
	branchDir := filepath.Join(b.rootPath(), name)
	rvs, err := os.ReadDir(branchDir)
	if err != nil {
		return false
	}
	for _, rv := range rvs {
		if !rv.IsDir() {
			continue
		}
		updates, err := os.ReadDir(filepath.Join(branchDir, rv.Name()))
		if err != nil {
			continue
		}
		for _, u := range updates {
			if !u.IsDir() {
				continue
			}
			updateDir := filepath.Join(branchDir, rv.Name(), u.Name())
			if _, err := os.Stat(filepath.Join(updateDir, ".check")); err == nil {
				return true
			}
			if _, err := os.Stat(filepath.Join(updateDir, "update-metadata.json")); err == nil {
				return true
			}
		}
	}
	return false
}

// MoveRootEntriesUnder copies every object whose key looks like it
// belongs to a v1 branch (see isV1BranchKey below) into the new
// {KeyPrefix}{appId}/ namespace, then deletes the source. Copy+delete is
// S3's only move primitive; each object copy is its own atomic API call,
// so interruptions leave the bucket in a consistent (maybe duplicated)
// state and re-running the migration converges.
func (b *S3Bucket) MoveRootEntriesUnder(appId string) error {
	client, err := services.GetS3Client()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	appPrefix := b.prefixedKey(appId + "/")

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.BucketName),
		Prefix: aws.String(b.KeyPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			key := *obj.Key
			if strings.HasPrefix(key, appPrefix) {
				// Already under the appId prefix — probably from a
				// previous partial migration run. Leave it.
				continue
			}
			relKey := strings.TrimPrefix(key, b.KeyPrefix)
			if relKey == ".migrationhistory" {
				continue
			}
			if !isV1BranchKey(relKey) {
				// Some other top-level entry (another appId, manual
				// data, tenant sharing the bucket). Don't assume it's
				// a branch to be re-parented.
				continue
			}
			newKey := appPrefix + relKey

			// CopySource must be URL-escaped; the SDK doesn't escape it
			// for us, and keys containing spaces/+/etc. silently fail the
			// copy otherwise.
			source := url.PathEscape(b.BucketName + "/" + key)
			if _, err := client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(b.BucketName),
				CopySource: aws.String(source),
				Key:        aws.String(newKey),
			}); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", key, newKey, err)
			}
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(b.BucketName),
				Key:    aws.String(key),
			}); err != nil {
				return fmt.Errorf("delete %s after copy: %w", key, err)
			}
		}
	}
	return nil
}

// isV1BranchKey returns true when a (keyPrefix-stripped) object key has
// the v1 shape {branch}/{runtimeVersion}/{updateId}/… — exactly 4
// non-empty segments, with the 4th being a file path. v2 keys have 5+
// segments ({appId}/{branch}/...), so this distinguishes them without a
// separate stat call. We do NOT parse updateId as numeric: users with
// custom update id schemes would be excluded from the migration for the
// wrong reason.
func isV1BranchKey(relKey string) bool {
	parts := strings.Split(relKey, "/")
	if len(parts) < 4 {
		return false
	}
	for i := range 4 {
		if parts[i] == "" {
			return false
		}
	}
	return true
}

// MoveRootEntriesUnder mirrors the S3 strategy on GCS: iterate with the
// existing prefix, copy each object to its new {appId}/-scoped key via
// CopierFrom, then delete the source. GCS's native rewrite handles
// cross-bucket and large-object semantics here since source and dest
// share the same bucket.
func (b *GCSBucket) MoveRootEntriesUnder(appId string) error {
	ctx := context.Background()
	bh, err := b.bucketHandle(ctx)
	if err != nil {
		return err
	}
	appPrefix := b.prefixedKey(appId + "/")

	it := bh.Objects(ctx, &storage.Query{Prefix: b.KeyPrefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}
		key := attrs.Name
		if strings.HasPrefix(key, appPrefix) {
			continue
		}
		relKey := strings.TrimPrefix(key, b.KeyPrefix)
		if relKey == ".migrationhistory" {
			continue
		}
		if !isV1BranchKey(relKey) {
			continue
		}
		newKey := appPrefix + relKey

		src := bh.Object(key)
		dst := bh.Object(newKey)
		if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
			return fmt.Errorf("copy %s -> %s: %w", key, newKey, err)
		}
		if err := src.Delete(ctx); err != nil {
			return fmt.Errorf("delete %s after copy: %w", key, err)
		}
	}
	return nil
}

// UnwrapBucket returns the underlying concrete backend when b is a
// validatingBucket decorator. External packages (the migration) need this
// so they can type-assert on *LocalBucket / *S3Bucket / *GCSBucket to
// call MoveRootEntriesUnder without going through the validating
// middleware (which would reject the root-level listing).
func UnwrapBucket(b Bucket) Bucket {
	if vb, ok := b.(*validatingBucket); ok {
		return vb.Inner
	}
	return b
}
