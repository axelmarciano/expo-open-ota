package bucket

import (
	"expo-open-ota/internal/types"
	"io"
)

// validatingBucket is a decorator around any Bucket implementation that
// validates every user-supplied identifier (branch, runtimeVersion, updateId,
// fileName, assetPath, migrationId) before delegating, rejecting values that
// contain path separators, "..", or that are empty. Mounted once in
// GetBucket(); concrete backends still assume inputs are clean but this layer
// guarantees that assumption even if a handler forgets to sanitize.
type validatingBucket struct {
	Inner Bucket
}

func (v *validatingBucket) GetBranches() ([]string, error) {
	return v.Inner.GetBranches()
}

func (v *validatingBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	if err := validateSegment("branch", branch); err != nil {
		return nil, err
	}
	return v.Inner.GetRuntimeVersions(branch)
}

func (v *validatingBucket) GetUpdates(branch, runtimeVersion string) ([]types.Update, error) {
	if err := validateSegment("branch", branch); err != nil {
		return nil, err
	}
	if err := validateSegment("runtimeVersion", runtimeVersion); err != nil {
		return nil, err
	}
	return v.Inner.GetUpdates(branch, runtimeVersion)
}

func (v *validatingBucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
	if err := validateUpdate(&update); err != nil {
		return nil, err
	}
	if err := validateRelativePath("assetPath", assetPath); err != nil {
		return nil, err
	}
	return v.Inner.GetFile(update, assetPath)
}

func (v *validatingBucket) RequestUploadUrlForFileUpdate(branch, runtimeVersion, updateId, fileName string) (string, error) {
	if err := validateSegment("branch", branch); err != nil {
		return "", err
	}
	if err := validateSegment("runtimeVersion", runtimeVersion); err != nil {
		return "", err
	}
	if err := validateSegment("updateId", updateId); err != nil {
		return "", err
	}
	if err := validateRelativePath("fileName", fileName); err != nil {
		return "", err
	}
	return v.Inner.RequestUploadUrlForFileUpdate(branch, runtimeVersion, updateId, fileName)
}

func (v *validatingBucket) UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error {
	if err := validateUpdate(&update); err != nil {
		return err
	}
	if err := validateRelativePath("fileName", fileName); err != nil {
		return err
	}
	return v.Inner.UploadFileIntoUpdate(update, fileName, file)
}

func (v *validatingBucket) DeleteUpdateFolder(branch, runtimeVersion, updateId string) error {
	if err := validateSegment("branch", branch); err != nil {
		return err
	}
	if err := validateSegment("runtimeVersion", runtimeVersion); err != nil {
		return err
	}
	if err := validateSegment("updateId", updateId); err != nil {
		return err
	}
	return v.Inner.DeleteUpdateFolder(branch, runtimeVersion, updateId)
}

func (v *validatingBucket) CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error) {
	if err := validateUpdate(previousUpdate); err != nil {
		return nil, err
	}
	if err := validateSegment("newUpdateId", newUpdateId); err != nil {
		return nil, err
	}
	return v.Inner.CreateUpdateFrom(previousUpdate, newUpdateId)
}

func (v *validatingBucket) RetrieveMigrationHistory() ([]string, error) {
	return v.Inner.RetrieveMigrationHistory()
}

func (v *validatingBucket) ApplyMigration(migrationId string) error {
	if err := validateSegment("migrationId", migrationId); err != nil {
		return err
	}
	return v.Inner.ApplyMigration(migrationId)
}

func (v *validatingBucket) RemoveMigrationFromHistory(migrationId string) error {
	if err := validateSegment("migrationId", migrationId); err != nil {
		return err
	}
	return v.Inner.RemoveMigrationFromHistory(migrationId)
}
