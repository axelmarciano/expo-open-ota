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
}

type UpdateType int

const (
	NormalUpdate UpdateType = iota
	Rollback
)

type ManifestAsset struct {
	Hash          string `json:"hash"`
	Key           string `json:"key"`
	FileExtension string `json:"fileExtension"`
	ContentType   string `json:"contentType"`
	Url           string `json:"url"`
}

type ExtraManifestData struct {
	ExpoClient json.RawMessage `json:"expoClient"`
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
	Environment    string
	RuntimeVersion string
	UpdateId       string
	CreatedAt      time.Duration
}

type BucketFile struct {
	Reader    io.ReadCloser
	CreatedAt time.Time
}
