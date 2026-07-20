// Package rollout implements the deterministic device bucketing behind progressive
// rollouts. Manifest and asset resolution share these functions so the two paths can
// never disagree about which cohort a device belongs to. No per-device state is stored:
// the bucket is a salted hash of the persistent EAS-Client-ID.
package rollout

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// Bucket maps a device to a stable position in [0, 1) for the given salt. An empty
// clientID returns 1.0 so a device that sends no EAS-Client-ID is never in a rollout.
func Bucket(clientID string, salt string) float64 {
	if clientID == "" {
		return 1.0
	}
	sum := sha256.Sum256([]byte(clientID + ":" + salt))
	return float64(binary.BigEndian.Uint64(sum[:8])) / float64(math.MaxUint64)
}

// InBucket reports whether the device falls inside a rollout serving percentage% of
// devices. Monotonic in percentage: a device inside at N stays inside at every M > N,
// so increasing a rollout never flips an already-served device back out.
func InBucket(clientID string, salt string, percentage int) bool {
	return Bucket(clientID, salt) < float64(percentage)/100
}

// UpdateSalt derives the fixed bucketing salt of a per-update rollout. Channel rollouts
// use the rollout row's UUID instead; the "update:" prefix keeps the two salt spaces
// structurally disjoint so the mechanisms draw independent cohorts.
func UpdateSalt(appId string, branchName string, runtimeVersion string, updateId string) string {
	return "update:" + appId + ":" + branchName + ":" + runtimeVersion + ":" + updateId
}
