package bucketmigration

import (
	"expo-open-ota/internal/bucket"
	"time"
)

type Migration interface {
	ID() string
	Timestamp() time.Time
	Up(b bucket.Bucket) error
	Down(b bucket.Bucket) error
}
