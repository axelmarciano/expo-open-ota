// Activation-time invariants of per-update rollouts. A rollout's control_update_id is
// captured by InsertUpdateWithRollout when upload URLs are handed out, but the rollout
// only goes live one bundle upload later, at MarkUpdateAsChecked. Everything here pins
// what must hold across that gap.
//
// Like the sibling rollout store tests, these need a real Postgres and skip without
// TEST_DATABASE_URL:
//
//	docker run -d --name eoo-pg -e POSTGRES_PASSWORD=test -p 55432:5432 postgres:16-alpine
//	TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/postgres?sslmode=disable" go test ./internal/store/
package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRolloutActivationRejectsStaleControl covers the update that was ALREADY uploading
// when a rollout publish started. It carries a lower id than the rollout, so an
// id-based supersede check does not see it, yet it reaches checked state later and
// moves the branch under the rollout.
//
// Left unguarded, the rollout activates with the control it captured at insert time,
// which now points one update too far back: the update in between answered 200 and
// printed "Update ready", but the out-of-bucket cohort is served the update before it
// and never receives it at all.
func TestRolloutActivationRejectsStaleControl(t *testing.T) {
	fixture := newRolloutFixture(t)
	ctx := context.Background()

	// The update everyone is running before either publish starts.
	fixture.createUpdate(t, rolloutTestDefaultBranch, 4000, "ios", true)

	// A plain publish calls requestUploadUrl: the row exists, unchecked, while its
	// bundle uploads.
	plainUpdate := fixture.createUpdate(t, rolloutTestDefaultBranch, 4100, "ios", false)

	// A rollout publish starts while that upload is still running. No rollout is active
	// yet, so it is allowed through, and its control is resolved here, against a branch
	// where 4100 is not yet visible.
	rolloutUpdate := fixture.createRolloutUpdate(t, rolloutTestDefaultBranch, 4200, "ios", 10)

	// The plain upload finishes first and is accepted: no rollout is active yet.
	require.NoError(t, fixture.updates.MarkUpdateAsChecked(ctx, plainUpdate))

	// The rollout upload finishes. Two outcomes are acceptable and this test stays
	// agnostic between them: the activation is refused because the branch moved under
	// it, or it activates with a control resolved at activation time. What must never
	// happen is activating with the control captured at insert time.
	rolloutErr := fixture.updates.MarkUpdateAsChecked(ctx, rolloutUpdate)

	envelope, err := fixture.updates.GetLatestUpdateWithRollout(ctx, fixture.appId, rolloutTestDefaultBranch, rolloutTestRuntime, "ios")
	require.NoError(t, err)
	require.NotNil(t, envelope)

	if rolloutErr != nil {
		assert.Equal(t, "4100", envelope.UpdateId, "the plain publish stays live when the rollout is refused")
		assert.Nil(t, envelope.RolloutPercentage, "no rollout may be active after a refused activation")
		return
	}

	require.Equal(t, "4200", envelope.UpdateId, "the rollout update is the one being served")
	require.NotNil(t, envelope.RolloutPercentage)
	require.Equal(t, 10, *envelope.RolloutPercentage)

	// The out-of-bucket cohort must receive the newest update that was live when the
	// rollout activated, which is the plain publish 4100.
	require.NotNil(t, envelope.Control, "out-of-bucket devices must have a control update")
	assert.Equal(t, "4100", envelope.Control.UpdateId,
		"control must reflect the branch at activation time; 4000 here means publish 4100 succeeded but is served to nobody")
}
