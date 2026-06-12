package branch

import (
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/providers"
)

func UpsertBranch(appId, branch string) error {
	branches, err := providers.FetchExpoBranches(appId)
	if err != nil {
		return err
	}
	if !helpers.StringInSlice(branch, branches) {
		return providers.CreateBranch(appId, branch)
	}
	return nil
}
