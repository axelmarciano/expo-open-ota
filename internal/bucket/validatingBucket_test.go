package bucket

import (
	"bytes"
	"expo-open-ota/internal/types"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubBucket records the last call so tests can verify whether the validating
// decorator delegated to the inner bucket or short-circuited on validation.
type stubBucket struct {
	called bool
}

func (s *stubBucket) mark() { s.called = true }

func (s *stubBucket) GetBranches() ([]string, error) { s.mark(); return nil, nil }
func (s *stubBucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	s.mark()
	return nil, nil
}
func (s *stubBucket) GetUpdates(branch, runtimeVersion string) ([]types.Update, error) {
	s.mark()
	return nil, nil
}
func (s *stubBucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
	s.mark()
	return nil, nil
}
func (s *stubBucket) RequestUploadUrlForFileUpdate(branch, runtimeVersion, updateId, fileName string) (string, error) {
	s.mark()
	return "", nil
}
func (s *stubBucket) UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error {
	s.mark()
	return nil
}
func (s *stubBucket) DeleteUpdateFolder(branch, runtimeVersion, updateId string) error {
	s.mark()
	return nil
}
func (s *stubBucket) CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error) {
	s.mark()
	return nil, nil
}
func (s *stubBucket) RetrieveMigrationHistory() ([]string, error)   { s.mark(); return nil, nil }
func (s *stubBucket) ApplyMigration(migrationId string) error       { s.mark(); return nil }
func (s *stubBucket) RemoveMigrationFromHistory(id string) error    { s.mark(); return nil }

func validUpdate() types.Update {
	return types.Update{Branch: "main", RuntimeVersion: "1.0", UpdateId: "123"}
}

func TestValidateSegment_RejectsTraversal(t *testing.T) {
	cases := []struct{ name, value string }{
		{"branch with dot-dot", ".."},
		{"branch with slash", "foo/bar"},
		{"branch with backslash", "foo\\bar"},
		{"empty", ""},
		{"single dot", "."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Error(t, validateSegment("branch", c.value))
		})
	}
}

func TestValidateSegment_AcceptsValidNames(t *testing.T) {
	cases := []string{"main", "feature-x", "v1.2.3", "release_2025", "..hidden"} // ".." as prefix of a name is allowed (not a segment of its own)
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			assert.NoError(t, validateSegment("branch", v))
		})
	}
}

func TestValidateRelativePath_RejectsTraversal(t *testing.T) {
	cases := []struct{ name, value string }{
		{"dot-dot segment", "assets/../../../etc/passwd"},
		{"leading dot-dot", "../secret"},
		{"absolute unix", "/etc/passwd"},
		{"absolute windows", "\\etc\\passwd"},
		{"empty", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Error(t, validateRelativePath("assetPath", c.value))
		})
	}
}

func TestValidateRelativePath_AcceptsNestedPaths(t *testing.T) {
	cases := []string{"image.png", "assets/img/logo.png", "deep/nested/path/file.js"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			assert.NoError(t, validateRelativePath("assetPath", v))
		})
	}
}

func TestValidatingBucket_GetFile_RejectsTraversalInBranch(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	update := types.Update{Branch: "../evil", RuntimeVersion: "1.0", UpdateId: "123"}
	_, err := v.GetFile(update, "asset.png")
	assert.Error(t, err)
	assert.False(t, stub.called, "inner bucket should not be called when validation fails")
}

func TestValidatingBucket_GetFile_RejectsTraversalInAssetPath(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	_, err := v.GetFile(validUpdate(), "../../../etc/passwd")
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_UploadFileIntoUpdate_RejectsTraversalInFileName(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	err := v.UploadFileIntoUpdate(validUpdate(), "../evil.js", bytes.NewReader(nil))
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_DeleteUpdateFolder_RejectsSlashInUpdateId(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	err := v.DeleteUpdateFolder("main", "1.0", "123/../456")
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_RequestUploadUrl_RejectsTraversalInFileName(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	_, err := v.RequestUploadUrlForFileUpdate("main", "1.0", "123", "../etc/passwd")
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_CreateUpdateFrom_RejectsTraversalInPreviousUpdate(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	prev := &types.Update{Branch: "../evil", RuntimeVersion: "1.0", UpdateId: "123"}
	_, err := v.CreateUpdateFrom(prev, "456")
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_ApplyMigration_RejectsSlash(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	err := v.ApplyMigration("../other")
	assert.Error(t, err)
	assert.False(t, stub.called)
}

func TestValidatingBucket_ValidInputsDelegate(t *testing.T) {
	stub := &stubBucket{}
	v := &validatingBucket{Inner: stub}
	_, err := v.GetFile(validUpdate(), "assets/image.png")
	assert.NoError(t, err)
	assert.True(t, stub.called)
}
