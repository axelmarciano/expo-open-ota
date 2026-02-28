package branch

import (
	"expo-open-ota/internal/bucket"
)

func FetchBranches() ([]string, error) {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.GetBranches()
}

func UpsertBranch(branch string) error {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.UpsertBranch(branch)
}
