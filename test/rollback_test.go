package test

import (
	"encoding/json"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/update"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func createRollbackRequest(projectRoot, branch, runtimeVersion, headerKey, headerValue, platform, commitHash string) (*httptest.ResponseRecorder, *mux.Router, *mux.Route, *http.Request) {
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	var q string
	if commitHash != "" {
		q = fmt.Sprintf("http://localhost:3000/rollback/%s?runtimeVersion=%s&platform=%s&commitHash=%s", branch, runtimeVersion, platform, commitHash)
	} else {
		q = fmt.Sprintf("http://localhost:3000/rollback/%s?runtimeVersion=%s&platform=%s", branch, runtimeVersion, platform)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{"APP_ID": "test-app-id", "BRANCH": branch})
	r.Header.Set(headerKey, headerValue)
	return w, mux.NewRouter(), nil, r
}

func TestToRollbackWithBadBearer(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	mockExpoForRequestUploadUrlTest("staging")
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer expo_bad_token", "ios", "hash")
	handlers.RollbackHandler(w, r)
	assert.Equal(t, 401, w.Code, "Expected status code 401")
	assert.Equal(t, "Error validating expo auth\n", w.Body.String(), "Expected error message")
}

func TestGoodRollback(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	mockExpoForRequestUploadUrlTest("staging")
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer expo_test_token", "ios", "hash")
	handlers.RollbackHandler(w, r)
	assert.Equal(t, 200, w.Code, "Expected status code 200")
	type Response struct {
		Branch         string `json:"branch"`
		RuntimeVersion string `json:"runtimeVersion"`
		UpdateId       string `json:"updateId"`
		CreatedAt      int64  `json:"createdAt"`
	}

	var body Response
	err = json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.NotEmpty(t, body.UpdateId, "Expected non-empty updateId")
	assert.NotEmpty(t, body.RuntimeVersion, "Expected non-empty runtimeVersion")
	assert.NotEmpty(t, body.Branch, "Expected non-empty branch")
	assert.NotEmpty(t, body.CreatedAt, "Expected non-empty createdAt")
	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion("test-app-id", "DO_NOT_USE", "1", "ios")
	if err != nil {
		t.Fatalf("Error getting latest update: %v", err)
	}
	assert.Equal(t, body.UpdateId, lastUpdate.UpdateId, "Expected updateId to match the latest update")
	updateType := update.GetUpdateType(*lastUpdate)
	assert.Equal(t, updateType, types.Rollback, "Expected update type to be rollback")
}

func TestGoodRollbackWithoutCommitHash(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	mockExpoForRequestUploadUrlTest("staging")
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer expo_test_token", "ios", "")
	handlers.RollbackHandler(w, r)
	assert.Equal(t, 200, w.Code, "Expected status code 200")
	type Response struct {
		Branch         string `json:"branch"`
		RuntimeVersion string `json:"runtimeVersion"`
		UpdateId       string `json:"updateId"`
		CreatedAt      int64  `json:"createdAt"`
	}

	var body Response
	err = json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.NotEmpty(t, body.UpdateId, "Expected non-empty updateId")
	assert.NotEmpty(t, body.RuntimeVersion, "Expected non-empty runtimeVersion")
	assert.NotEmpty(t, body.Branch, "Expected non-empty branch")
	assert.NotEmpty(t, body.CreatedAt, "Expected non-empty createdAt")
	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion("test-app-id", "DO_NOT_USE", "1", "ios")
	if err != nil {
		t.Fatalf("Error getting latest update: %v", err)
	}
	assert.Equal(t, body.UpdateId, lastUpdate.UpdateId, "Expected updateId to match the latest update")
	updateType := update.GetUpdateType(*lastUpdate)
	assert.Equal(t, updateType, types.Rollback, "Expected update type to be rollback")
}

// TestRollbackDoesNotPoisonLatestUpdateCache is a regression test for the
// race in MarkUpdateAsChecked where cache.Delete used to run before the
// .check file was written.
//
// The race: between the cache invalidation and the .check write there was
// a wide window (StoreUpdateUUIDInMetadata does slow bucket I/O). A
// concurrent /manifest request in that window would miss the cache, scan
// the bucket, see the new update on disk WITHOUT a .check file, filter it
// out via IsUpdateValid, fall back to the previous update, and re-cache
// it under the lastUpdate key for the full 1800s TTL — serving a stale
// manifest for up to 30 minutes after the publish/rollback.
//
// The fix reorders MarkUpdateAsChecked to write .check before deleting
// the cache, so the bad intermediate state (new-update-without-check +
// empty-cache) is never visible to concurrent readers.
//
// This test hammers GetLatestUpdateBundlePathForRuntimeVersion from many
// goroutines while a rollback is in flight, then asserts the final cache
// state points at the rollback and not at the pre-existing fixture.
func TestRollbackDoesNotPoisonLatestUpdateCache(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	mockExpoForRequestUploadUrlTest("staging")
	projectRoot, err := findProjectRoot()
	require.NoError(t, err)

	const appId = "test-app-id"
	const branch = "branch-1"
	const runtimeVersion = "1"
	const platform = "android"
	const fixtureUpdateId = "1674170951" // pre-existing fixture we expect to be superseded

	// Warm the lastUpdate cache with the fixture so a poisoned cache can
	// actually point at the stale value.
	warmed, err := update.GetLatestUpdateBundlePathForRuntimeVersion(appId, branch, runtimeVersion, platform)
	require.NoError(t, err)
	require.NotNil(t, warmed)
	require.Equal(t, fixtureUpdateId, warmed.UpdateId, "fixture setup: expected cache warmed with pre-existing update")

	// Hammer the read path from many goroutines. The old ordering would
	// have given one of these readers a chance to re-cache the fixture
	// mid-rollback. Keep going past the rollback handler's return so any
	// poisoned value has to survive to the final assertion.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var reads int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = update.GetLatestUpdateBundlePathForRuntimeVersion(appId, branch, runtimeVersion, platform)
					atomic.AddInt64(&reads, 1)
				}
			}
		}()
	}

	// Fire the rollback while readers are hammering.
	w, _, _, r := createRollbackRequest(projectRoot, branch, runtimeVersion, "Authorization", "Bearer expo_test_token", platform, "hash")
	handlers.RollbackHandler(w, r)
	require.Equal(t, http.StatusOK, w.Code, "rollback handler failed: %s", w.Body.String())

	close(stop)
	wg.Wait()

	// Definitive assertion: after the rollback, the latest update must be
	// the rollback itself, not the pre-existing fixture. A failure here
	// means a concurrent reader poisoned the cache with the stale fixture.
	latest, err := update.GetLatestUpdateBundlePathForRuntimeVersion(appId, branch, runtimeVersion, platform)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.NotEqual(t, fixtureUpdateId, latest.UpdateId, "lastUpdate cache poisoned with pre-rollback fixture after %d reads", reads)
	assert.Equal(t, types.Rollback, update.GetUpdateType(*latest), "expected latest update to be a rollback")
}
