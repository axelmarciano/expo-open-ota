package helpers

import (
	"time"
)

// NormalizeTimestamp converts both legacy millisecond timestamps
// and new packed platform timestamps into a standard UTC time.Time object.
func NormalizeTimestamp(value int64) time.Time {
	if value > 9999999999999 {
		return time.UnixMilli(value / 10).UTC()
	}
	return time.UnixMilli(value).UTC()
}

// NormalizeTimestampToDuration extracts the time from both old and new formats
// and returns it as a time.Duration since the Unix Epoch.
func NormalizeTimestampToDuration(value int64) time.Duration {
	epoch := time.Unix(0, 0).UTC()
	return NormalizeTimestamp(value).Sub(epoch)
}
