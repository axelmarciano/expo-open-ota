package branch

import (
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/services"
)

func UpsertBranch(appId, branch string) error {
	branches, err := services.FetchExpoBranches(appId)
	if err != nil {
		return err
	}
	if !helpers.StringInSlice(branch, branches) {
		return services.CreateBranch(appId, branch)
	}
	return nil
}
