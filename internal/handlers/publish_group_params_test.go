// The wire contract of the publishGroup parameter, which has two distinct
// readings: a STAMP on the publish route (ignorable: worst case the rows list
// ungrouped) and a TARGET on the rollback/republish routes (never ignorable:
// it selects what the operation acts on).
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func publishGroupRequest(t *testing.T, rawValue string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/app/requestUploadUrl/main", nil)
	if rawValue != "" {
		query := request.URL.Query()
		query.Set("publishGroup", rawValue)
		request.URL.RawQuery = query.Encode()
	}
	return request
}

func TestParsePublishGroupStampControlPlane(t *testing.T) {
	t.Setenv("DB_URL", "postgres://test")

	parsed, err := parsePublishGroup(publishGroupRequest(t, "8A3C66C1-8D66-44BC-A03C-2C4C5F4B2E7B"))
	require.NoError(t, err)
	require.NotNil(t, parsed)
	// Normalized: the row and its acknowledgment always carry the canonical form.
	assert.Equal(t, "8a3c66c1-8d66-44bc-a03c-2c4c5f4b2e7b", *parsed)

	_, err = parsePublishGroup(publishGroupRequest(t, "not-a-uuid"))
	require.Error(t, err)

	parsed, err = parsePublishGroup(publishGroupRequest(t, ""))
	require.NoError(t, err)
	assert.Nil(t, parsed)
}

func TestParsePublishGroupStampStateless(t *testing.T) {
	t.Setenv("DB_URL", "")

	// The stamp is ignored entirely in stateless mode, malformed included: the
	// missing acknowledgment is what tells the CLI nothing was grouped.
	for _, rawValue := range []string{"8a3c66c1-8d66-44bc-a03c-2c4c5f4b2e7b", "not-a-uuid", ""} {
		parsed, err := parsePublishGroup(publishGroupRequest(t, rawValue))
		require.NoError(t, err, rawValue)
		assert.Nil(t, parsed, rawValue)
	}
}

func TestParsePublishGroupTargetControlPlane(t *testing.T) {
	t.Setenv("DB_URL", "postgres://test")

	parsed, err := parsePublishGroupTarget(publishGroupRequest(t, "8A3C66C1-8D66-44BC-A03C-2C4C5F4B2E7B"))
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "8a3c66c1-8d66-44bc-a03c-2c4c5f4b2e7b", *parsed)

	_, err = parsePublishGroupTarget(publishGroupRequest(t, "not-a-uuid"))
	require.Error(t, err)

	parsed, err = parsePublishGroupTarget(publishGroupRequest(t, ""))
	require.NoError(t, err)
	assert.Nil(t, parsed)
}

func TestParsePublishGroupTargetStateless(t *testing.T) {
	t.Setenv("DB_URL", "")

	// A target cannot be silently dropped: ignoring it would run a DIFFERENT
	// operation than the one requested, so stateless mode refuses outright.
	for _, rawValue := range []string{"8a3c66c1-8d66-44bc-a03c-2c4c5f4b2e7b", "not-a-uuid"} {
		_, err := parsePublishGroupTarget(publishGroupRequest(t, rawValue))
		require.Error(t, err, rawValue)
	}

	// Absent stays absent: the historical single-target flows are untouched.
	parsed, err := parsePublishGroupTarget(publishGroupRequest(t, ""))
	require.NoError(t, err)
	assert.Nil(t, parsed)
}
