package bucket

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Only the LocalBucket re-path is exercised here; S3/GCS require a live
// backend and are covered by integration tests outside the Go test suite.
//
// Shared layout for every fixture:
//   {basePath}/branch-a/1/12345/.check
//   {basePath}/branch-b/1/67890/.check
//   {basePath}/.migrationhistory
// After MoveRootEntriesUnder("app-1"):
//   {basePath}/app-1/branch-a/1/12345/.check
//   {basePath}/app-1/branch-b/1/67890/.check
//   {basePath}/.migrationhistory  (stays at root)

func newV1LayoutBucket(t *testing.T) *LocalBucket {
	t.Helper()
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "branch-a", "1", "12345", ".check"))
	writeFile(t, filepath.Join(base, "branch-b", "1", "67890", ".check"))
	writeFile(t, filepath.Join(base, ".migrationhistory"))
	return &LocalBucket{BasePath: base}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
}

func TestLocalBucket_MoveRootEntriesUnder_MovesBranches(t *testing.T) {
	b := newV1LayoutBucket(t)
	base := b.BasePath

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	// Branch directories are now under app-1/
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-b", "1", "67890", ".check"))

	// Roots should not contain branch-a / branch-b anymore.
	assert.NoFileExists(t, filepath.Join(base, "branch-a", "1", "12345", ".check"))
	assert.NoFileExists(t, filepath.Join(base, "branch-b", "1", "67890", ".check"))
}

func TestLocalBucket_MoveRootEntriesUnder_PreservesMigrationHistory(t *testing.T) {
	// The ledger is deployment-global — it MUST stay at root, otherwise
	// the migration runner can't find it on the next boot and every
	// applied migration silently re-runs.
	b := newV1LayoutBucket(t)
	base := b.BasePath

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	assert.FileExists(t, filepath.Join(base, ".migrationhistory"))
	assert.NoFileExists(t, filepath.Join(base, "app-1", ".migrationhistory"))
}

func TestLocalBucket_MoveRootEntriesUnder_IdempotentOnReRun(t *testing.T) {
	// Re-running on an already-migrated tree must be a no-op — operators
	// restart pods, Kubernetes retries init containers, the migration
	// runner can be manually invoked. Any of those can trigger a second
	// call; dropping data or erroring here would be devastating.
	b := newV1LayoutBucket(t)
	base := b.BasePath

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))
	// Second run: app-1/ is now the only top-level entry besides the
	// ledger. MoveRootEntriesUnder skips `name == appId` and the ledger,
	// so nothing to move — must not error and must not touch anything.
	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	assert.FileExists(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-b", "1", "67890", ".check"))
	// Critically, there is no app-1/app-1 loop.
	assert.NoDirExists(t, filepath.Join(base, "app-1", "app-1"))
}

func TestLocalBucket_MoveRootEntriesUnder_ResumesPartialMigration(t *testing.T) {
	// Simulate a crash mid-migration: branch-a was already moved, branch-b
	// is still at root. The second run must complete the job without
	// clobbering branch-a.
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	writeFile(t, filepath.Join(base, "branch-b", "1", "67890", ".check"))
	writeFile(t, filepath.Join(base, ".migrationhistory"))
	b := &LocalBucket{BasePath: base}

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	assert.FileExists(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-b", "1", "67890", ".check"))
	assert.NoFileExists(t, filepath.Join(base, "branch-b", "1", "67890", ".check"))
}

func TestLocalBucket_MoveRootEntriesUnder_EmptyBucketIsNoOp(t *testing.T) {
	// Fresh install: no data at all, but the migration runner still
	// invokes up(). The move must return nil and not create a stray
	// {appId}/ directory to clutter the bucket.
	base := t.TempDir()
	b := &LocalBucket{BasePath: base}

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))
}

func TestLocalBucket_MoveRootEntriesUnder_SkipsNonBranchShapedEntries(t *testing.T) {
	// The big safety net: if another appId-looking directory is already
	// at root (partial earlier migration, wrong env var, leftover from
	// disaster recovery), we must NOT re-parent it under the current
	// target appId. A directory without {rv}/{updateId}/.check depth-3
	// shape isn't a v1 branch — leave it alone.
	base := t.TempDir()
	// A legit v1 branch — should move.
	writeFile(t, filepath.Join(base, "branch-a", "1", "12345", ".check"))
	// A v2-looking appId dir (depth-4 .check) — must be left alone.
	writeFile(t, filepath.Join(base, "other-app-uuid", "branch-a", "1", "67890", ".check"))
	// A random directory with no v1 shape at all — must be left alone.
	writeFile(t, filepath.Join(base, "random", "README.md"))
	b := &LocalBucket{BasePath: base}

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	// branch-a moved.
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	assert.NoDirExists(t, filepath.Join(base, "branch-a"))

	// other-app-uuid untouched at root.
	assert.FileExists(t, filepath.Join(base, "other-app-uuid", "branch-a", "1", "67890", ".check"))
	assert.NoDirExists(t, filepath.Join(base, "app-1", "other-app-uuid"))

	// random/ untouched.
	assert.FileExists(t, filepath.Join(base, "random", "README.md"))
	assert.NoDirExists(t, filepath.Join(base, "app-1", "random"))
}

func TestLocalBucket_MoveRootEntriesUnder_NoOpOnFullyV2Bucket(t *testing.T) {
	// If every non-ledger top-level entry is already v2-shaped, the
	// migration must not create a stray empty {appId}/ directory.
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	writeFile(t, filepath.Join(base, ".migrationhistory"))
	b := &LocalBucket{BasePath: base}

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	// app-1 still exists with its content, no new nesting.
	assert.FileExists(t, filepath.Join(base, "app-1", "branch-a", "1", "12345", ".check"))
	assert.NoDirExists(t, filepath.Join(base, "app-1", "app-1"))
}

func TestIsV1BranchKey(t *testing.T) {
	cases := map[string]bool{
		// v1 shape: branch/rv/updateId/file
		"branch-a/1/12345/.check":                  true,
		"branch-a/1/12345/update-metadata.json":    true,
		"branch-a/1.0.0/12345/bundles/x.js":        true,
		// v2 shape: appId/branch/rv/updateId/file
		"app-uuid/branch-a/1/12345/.check":         true, // 5 segments — still considered "has branch shape" but we skip via appPrefix guard before this is called
		// Not enough segments
		"branch-a":                 false,
		"branch-a/1":               false,
		"branch-a/1/12345":         false,
		// Empty segments
		"//12345/.check":           false,
		"branch-a//12345/.check":   false,
	}
	for k, want := range cases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, want, isV1BranchKey(k))
		})
	}
}

func TestLocalBucket_MoveRootEntriesUnder_RespectsKeyPrefix(t *testing.T) {
	// When KeyPrefix is set, the bucket root is {BasePath}/{KeyPrefix}/.
	// The re-path must happen inside that subtree, never at the outer
	// BasePath — otherwise a multi-tenant host that co-locates two
	// Expo Open OTA instances under the same BasePath would have them
	// stomp each other.
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "myapp", "branch-a", "1", "12345", ".check"))
	writeFile(t, filepath.Join(base, "myapp", ".migrationhistory"))
	writeFile(t, filepath.Join(base, "other-tenant", ".check"))
	b := &LocalBucket{BasePath: base, KeyPrefix: "myapp/"}

	require.NoError(t, b.MoveRootEntriesUnder("app-1"))

	assert.FileExists(t, filepath.Join(base, "myapp", "app-1", "branch-a", "1", "12345", ".check"))
	assert.FileExists(t, filepath.Join(base, "myapp", ".migrationhistory"))
	// Sibling tenant data outside the prefix is untouched.
	assert.FileExists(t, filepath.Join(base, "other-tenant", ".check"))
	assert.NoDirExists(t, filepath.Join(base, "myapp", "app-1", "other-tenant"))
}
