package _0250417_persist_uuid

import (
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/migration"
	"time"
)

// 20250417_persist_uuid was written against the v1 single-app bucket layout
// ({prefix}/{branch}/{runtimeVersion}/{updateId}). The v2 multi-app layout
// ({prefix}/{appId}/{branch}/...) breaks the iteration contract this
// migration relied on — there is no bucket-level ListApps() to rebuild the
// outer loop from. Since v2 is a breaking change with no in-place upgrade,
// any data that required this migration was already processed by the v1
// server before cutover; on a fresh v2 install there is nothing to persist.
// Kept as a no-op stub to preserve the migration ledger entry.
func init() {
	migration.Register(migration.BaseMigration{
		Id:       "20250417_persist_uuid",
		Time:     time.Date(2025, 4, 17, 0, 0, 0, 0, time.UTC),
		UpFunc:   func(b bucket.Bucket) error { return nil },
		DownFunc: func(b bucket.Bucket) error { return nil },
	})
}
