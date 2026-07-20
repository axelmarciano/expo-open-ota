package test

import (
	"errors"
	"expo-open-ota/internal/types"
	"expo-open-ota/internal/update"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A resolved update whose metadata.json is absent from storage (phantom row
// after a storage switch, partial upload) must surface as a typed error, not
// as an empty metadata that later masquerades as "platform not supported".
func TestGetMetadataMissingReturnsSentinel(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	phantom := types.Update{AppId: "test-app-id", Branch: "branch-1", RuntimeVersion: "1", UpdateId: "999999"}
	_, err := update.GetMetadata(phantom)
	assert.True(t, errors.Is(err, update.ErrUpdateMetadataMissing), "expected ErrUpdateMetadataMissing, got %v", err)
	assert.Contains(t, err.Error(), "test-app-id/branch-1/1/999999")
}

// Comparing against an update without metadata (rollback folder, phantom)
// must answer "not identical" instead of failing the publish flow.
func TestAreUpdatesIdenticalToleratesMissingMetadata(t *testing.T) {
	teardown := setup(t)
	defer teardown()
	real := types.Update{AppId: "test-app-id", Branch: "branch-1", RuntimeVersion: "1", UpdateId: "1674170951"}
	phantom := types.Update{AppId: "test-app-id", Branch: "branch-1", RuntimeVersion: "1", UpdateId: "999999"}

	identical, err := update.AreUpdatesIdentical(real, phantom)
	assert.Nil(t, err)
	assert.False(t, identical)

	identical, err = update.AreUpdatesIdentical(phantom, real)
	assert.Nil(t, err)
	assert.False(t, identical)
}
