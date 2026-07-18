package types

import (
	"encoding/json"
	"io"
	"time"
)

type Asset struct {
	Path string `json:"path"`
	Ext  string `json:"ext"`
}

type PlatformMetadata struct {
	Bundle string  `json:"bundle"`
	Assets []Asset `json:"assets"`
}

type FileMetadata struct {
	Android PlatformMetadata `json:"android"`
	IOS     PlatformMetadata `json:"ios"`
}

type MetadataObject struct {
	Version      int          `json:"version"`
	Bundler      string       `json:"bundler"`
	FileMetadata FileMetadata `json:"fileMetadata"`
}

type UpdateMetadata struct {
	MetadataJSON MetadataObject `json:"metadataJSON"`
	CreatedAt    string         `json:"createdAt"`
	ID           string         `json:"id"`
	Fingerprint  string         `json:"fingerprint"`
}

type UpdateItem struct {
	UpdateUUID string `json:"updateUUID"`
	UpdateId   string `json:"updateId"`
	CreatedAt  string `json:"createdAt"`
	CommitHash string `json:"commitHash"`
	Platform   string `json:"platform"`
	Message    string `json:"message,omitempty"`
}

type UpdateStoredMetadata struct {
	Platform   string `json:"platform"`
	CommitHash string `json:"commitHash"`
	UpdateUUID string `json:"updateUUID"`
	Message    string `json:"message,omitempty"`
}

type UpdateType int

const (
	NormalUpdate UpdateType = iota
	Rollback
)

type UpdateDetails struct {
	UpdateUUID string     `json:"updateUUID"`
	UpdateId   string     `json:"updateId"`
	CreatedAt  string     `json:"createdAt"`
	CommitHash string     `json:"commitHash"`
	Platform   string     `json:"platform"`
	Message    string     `json:"message,omitempty"`
	Type       UpdateType `json:"type"`
	ExpoConfig string     `json:"expoConfig"`
}

type ApiKeyMetadata struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Hint       string  `json:"hint"`
	CreatedAt  string  `json:"createdAt"`
	LastUsedAt *string `json:"lastUsedAt,omitempty"`
}

type ManifestAsset struct {
	Hash          string `json:"hash"`
	Key           string `json:"key"`
	FileExtension string `json:"fileExtension"`
	ContentType   string `json:"contentType"`
	Url           string `json:"url"`
}

type ExtraManifestData struct {
	ExpoClient json.RawMessage `json:"expoClient"`
	Branch     string          `json:"branch"`
}

type UpdateManifest struct {
	Id             string            `json:"id"`
	CreatedAt      string            `json:"createdAt"`
	RunTimeVersion string            `json:"runtimeVersion"`
	Metadata       json.RawMessage   `json:"metadata"`
	Assets         []ManifestAsset   `json:"assets"`
	LaunchAsset    ManifestAsset     `json:"launchAsset"`
	Extra          ExtraManifestData `json:"extra"`
}

type RollbackDirectiveParameters struct {
	CommitTime string `json:"commitTime"`
}

type RollbackDirective struct {
	Type       string                      `json:"type"`
	Parameters RollbackDirectiveParameters `json:"parameters"`
}

type NoUpdateAvailableDirective struct {
	Type string `json:"type"`
}

type Update struct {
	AppId          string        `json:"appId"`
	Branch         string        `json:"branch"`
	RuntimeVersion string        `json:"runtimeVersion"`
	UpdateId       string        `json:"updateId"`
	CreatedAt      time.Duration `json:"createdAt"`
}

type ChannelMapping struct {
	ReleaseChannelName string  `json:"releaseChannelName"`
	ReleaseChannelId   string  `json:"releaseChannelId"`
	BranchName         *string `json:"branchName"`
	BranchId           *string `json:"branchId"`
	CreatedAt          *string `json:"createdAt"`
}

type BranchMapping struct {
	BranchName     string  `json:"branchName"`
	BranchId       *string `json:"branchId"`
	ReleaseChannel *string `json:"releaseChannel"`
	CreatedAt      *string `json:"createdAt"`
	// Enterprise branch protection; always false in stateless mode.
	Protected bool `json:"protected"`
}

type RuntimeVersionWithStats struct {
	RuntimeVersion  string `json:"runtimeVersion"`
	LastUpdatedAt   string `json:"lastUpdatedAt"`
	CreatedAt       string `json:"createdAt"`
	NumberOfUpdates int    `json:"numberOfUpdates"`
}

type BucketFile struct {
	Reader    io.ReadCloser
	CreatedAt time.Time
}

type Auth struct {
	Token         *string
	SessionSecret *string
}
