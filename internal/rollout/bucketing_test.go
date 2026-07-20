package rollout

import (
	"fmt"
	"math"
	"testing"

	"github.com/google/uuid"
)

func makeClientIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	return ids
}

func TestBucketIsDeterministic(t *testing.T) {
	clientID := uuid.NewString()
	salt := uuid.NewString()
	first := Bucket(clientID, salt)
	for i := 0; i < 10; i++ {
		if got := Bucket(clientID, salt); got != first {
			t.Fatalf("Bucket is not deterministic: got %v then %v", first, got)
		}
	}
	if first < 0 || first >= 1 {
		t.Fatalf("Bucket out of [0, 1): %v", first)
	}
}

func TestEmptyClientIDNeverInRollout(t *testing.T) {
	for _, pct := range []int{1, 10, 50, 99, 100} {
		if InBucket("", "any-salt", pct) {
			t.Fatalf("empty clientID must never be in rollout (pct=%d)", pct)
		}
	}
	if got := Bucket("", "any-salt"); got != 1.0 {
		t.Fatalf("empty clientID bucket must be 1.0, got %v", got)
	}
}

func TestDistributionMatchesPercentage(t *testing.T) {
	const sampleSize = 100000
	ids := makeClientIDs(sampleSize)
	salt := uuid.NewString()
	for _, pct := range []int{1, 10, 50, 99} {
		t.Run(fmt.Sprintf("pct=%d", pct), func(t *testing.T) {
			inCount := 0
			for _, id := range ids {
				if InBucket(id, salt, pct) {
					inCount++
				}
			}
			observed := float64(inCount) / float64(sampleSize) * 100
			if math.Abs(observed-float64(pct)) > 1 {
				t.Fatalf("observed %.2f%% in bucket, expected %d%% within 1 point", observed, pct)
			}
		})
	}
}

// TestMonotonicStickiness pins the property progression relies on: a device served at
// percentage N must stay served at every higher percentage, for the same salt.
func TestMonotonicStickiness(t *testing.T) {
	ids := makeClientIDs(1000)
	salt := uuid.NewString()
	for _, id := range ids {
		wasIn := false
		for pct := 1; pct <= 99; pct++ {
			in := InBucket(id, salt, pct)
			if wasIn && !in {
				t.Fatalf("device %s left the rollout when percentage increased to %d", id, pct)
			}
			wasIn = in
		}
	}
}

// TestSaltIndependence checks that two rollouts with different salts draw independent
// cohorts: the overlap of two 30% rollouts should be about 9% (0.3 * 0.3), not 30%
// (which would mean the same devices are always picked first).
func TestSaltIndependence(t *testing.T) {
	const sampleSize = 10000
	ids := makeClientIDs(sampleSize)
	saltA := uuid.NewString()
	saltB := uuid.NewString()
	overlap := 0
	for _, id := range ids {
		if InBucket(id, saltA, 30) && InBucket(id, saltB, 30) {
			overlap++
		}
	}
	observed := float64(overlap) / float64(sampleSize) * 100
	if math.Abs(observed-9) > 2 {
		t.Fatalf("observed %.2f%% overlap between two 30%% rollouts, expected about 9%% within 2 points", observed)
	}
}
