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
	r = mux.SetURLVars(r, map[string]string{"BRANCH": branch})
	r.Header.Set(headerKey, headerValue)
	return w, mux.NewRouter(), nil, r
}

func TestToRollbackWithBadBearer(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer expo_bad_token", "ios", "hash")
	handlers.RollbackHandler(w, r)
	assert.Equal(t, 401, w.Code, "Expected status code 401")
	assert.Equal(t, "Unauthorized\n", w.Body.String(), "Expected error message")
}

func TestGoodRollback(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer EOAS_API_KEY", "ios", "hash")
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
	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion("DO_NOT_USE", "1", "ios")
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
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Error finding project root: %v", err)
	}
	w, _, _, r := createRollbackRequest(projectRoot, "DO_NOT_USE", "1", "Authorization", "Bearer EOAS_API_KEY", "ios", "")
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
	lastUpdate, err := update.GetLatestUpdateBundlePathForRuntimeVersion("DO_NOT_USE", "1", "ios")
	if err != nil {
		t.Fatalf("Error getting latest update: %v", err)
	}
	assert.Equal(t, body.UpdateId, lastUpdate.UpdateId, "Expected updateId to match the latest update")
	updateType := update.GetUpdateType(*lastUpdate)
	assert.Equal(t, updateType, types.Rollback, "Expected update type to be rollback")
}
