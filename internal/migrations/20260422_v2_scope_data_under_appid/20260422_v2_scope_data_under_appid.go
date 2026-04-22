package _0260422_v2_scope_data_under_appid

import (
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/migration"
	"fmt"
	"log"
	"os"
	"time"
)

// 20260422_v2_scope_data_under_appid moves v1 bucket data into the v2
// {appId}-scoped layout exactly once on boot. v1 stored updates under
// {prefix}/{branch}/{runtimeVersion}/{updateId}/…; v2 requires
// {prefix}/{appId}/{branch}/…. Without this migration, a v1 deploy that
// upgrades in place loses visibility of every previously published update.
//
// Guards:
//   - Only runs on the single-app flat-env path (EXPO_APP_ID set,
//     EXPO_APPS_JSON NOT set). Multi-app upgrades are inherently
//     ambiguous — the server cannot know which v1 branch belonged to
//     which configured app — so they must be re-pathed manually per the
//     migration guide.
//   - Operator can opt out with SKIP_V1_TO_V2_BUCKET_MIGRATION=true if
//     they have already re-pathed the data themselves or want to
//     schedule the move separately.
//
// The move is driven by the validated Bucket's underlying concrete
// backend (via bucket.UnwrapBucket + type assertion on
// *LocalBucket / *S3Bucket / *GCSBucket) because the validating
// decorator rejects root-level listing — it expects scoped appId args.
func init() {
	migration.Register(migration.BaseMigration{
		Id:       "20260422_v2_scope_data_under_appid",
		Time:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		UpFunc:   up,
		DownFunc: func(b bucket.Bucket) error { return nil },
	})
}

func up(b bucket.Bucket) error {
	if os.Getenv("SKIP_V1_TO_V2_BUCKET_MIGRATION") == "true" {
		log.Println("⏩ SKIP_V1_TO_V2_BUCKET_MIGRATION=true — skipping bucket re-path.")
		return nil
	}
	// Multi-app deployments use EXPO_APPS_JSON; we cannot guess which v1
	// branch belongs to which configured app, so this migration is a
	// no-op in that case. Operators must re-path manually.
	if os.Getenv("EXPO_APPS_JSON") != "" {
		log.Println("⏩ Multi-app config (EXPO_APPS_JSON) detected — v1-to-v2 bucket migration is manual, skipping.")
		return nil
	}
	appId := os.Getenv("EXPO_APP_ID")
	if appId == "" {
		return nil
	}

	inner := bucket.UnwrapBucket(b)
	log.Printf("🧱 v1→v2 bucket re-path: moving root entries under %q …", appId)
	switch concrete := inner.(type) {
	case *bucket.LocalBucket:
		if err := concrete.MoveRootEntriesUnder(appId); err != nil {
			return fmt.Errorf("LocalBucket re-path: %w", err)
		}
	case *bucket.S3Bucket:
		if err := concrete.MoveRootEntriesUnder(appId); err != nil {
			return fmt.Errorf("S3Bucket re-path: %w", err)
		}
	case *bucket.GCSBucket:
		if err := concrete.MoveRootEntriesUnder(appId); err != nil {
			return fmt.Errorf("GCSBucket re-path: %w", err)
		}
	default:
		return fmt.Errorf("unsupported bucket backend for v1-to-v2 migration: %T", inner)
	}
	log.Println("✅ v1→v2 bucket re-path complete.")
	return nil
}
