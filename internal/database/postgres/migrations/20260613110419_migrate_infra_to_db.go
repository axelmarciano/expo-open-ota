package migrations

import (
	"context"
	"database/sql"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/helpers"
	"expo-open-ota/internal/store"
	"expo-open-ota/internal/types"
	update2 "expo-open-ota/internal/update"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pressly/goose/v3"
)

var dbEngine *database.Engine

func SetEngine(e *database.Engine) {
	dbEngine = e
}

func init() {
	goose.AddMigrationContext(UpMigrateEnvJSON, DownMigrateEnvJSON)
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func parseRFC3339ToTz(str string, fieldName string) pgtype.Timestamptz {
	if parsedTime, err := time.Parse(time.RFC3339, str); err == nil {
		return toTimestamptz(parsedTime.UTC())
	}
	ms, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		log.Printf("⚠️ Warning: Malformed time string for %s ('%s'), falling back to Now: %v", fieldName, str, err)
		return toTimestamptz(time.Now())
	}
	normalizedTime := helpers.NormalizeTimestamp(ms)
	return toTimestamptz(normalizedTime)
}

func UpMigrateEnvJSON(ctx context.Context, tx *sql.Tx) error {
	log.Println("🚀 [DATABASE] Starting migration of infrastructure from env/bucket to the database...")
	config.LoadApps()
	apps, source, err := config.ReadApps()
	if err != nil {
		log.Printf("Error reading apps from config: %v", err)
		return err
	}
	if len(apps) == 0 {
		log.Printf("No apps found in config source '%s'", source)
		return nil
	}

	err = dbEngine.WithTx(ctx, func(qtx *pgdb.Queries) error {
		resolvedBucket := bucket.GetBucket()
		branchStore := store.NewBucketBranchStore(resolvedBucket)
		channelStore := store.NewBucketChannelStore(resolvedBucket)
		updateStore := store.NewBucketUpdateStore(resolvedBucket)

		for _, app := range apps {
			if _, err := uuid.Parse(app.Id); err != nil {
				log.Printf("Error parsing app ID '%s': %v", app.Id, err)
				return err
			}

			params := pgdb.MigrateLegacyAppParams{
				ID:                 store.ToPgUUID(app.Id),
				Name:               app.Name,
				KeysMode:           helpers.StringOrNil(string(app.Keys.Mode)),
				SealedPublicKey:    helpers.StringOrNil(string(app.Keys.SealedPublicKey)),
				SealedPrivateKey:   helpers.StringOrNil(string(app.Keys.SealedPrivateKey)),
				PathPublicKey:      helpers.StringOrNil(string(app.Keys.PublicPath)),
				PathPrivateKey:     helpers.StringOrNil(string(app.Keys.PrivatePath)),
				AwsSecretIDPublic:  helpers.StringOrNil(string(app.Keys.PublicSecretId)),
				AwsSecretIDPrivate: helpers.StringOrNil(string(app.Keys.PrivateSecretId)),
			}

			if err := qtx.MigrateLegacyApp(ctx, params); err != nil {
				log.Printf("Error inserting app '%s' into database: %v", app.Id, err)
				return err
			}

			// Fetch branches for the app
			branches, err := branchStore.GetBranches(ctx, app.Id)
			if err != nil {
				log.Printf("Error fetching branches for app '%s': %v", app.Id, err)
				return err
			}

			channelNameToBranchId := make(map[string]int64)
			insertedRuntimeVersions := make(map[string]bool)

			for _, branch := range branches {
				branchId, err := qtx.MigrateLegacyBranch(ctx, pgdb.MigrateLegacyBranchParams{
					AppID: store.ToPgUUID(app.Id),
					Name:  branch.BranchName,
				})
				if err != nil {
					log.Printf("Error inserting branch '%s' for app '%s': %v", branch.BranchName, app.Id, err)
					return err
				}

				if branch.ReleaseChannel != nil && *branch.ReleaseChannel != "" {
					channelNameToBranchId[*branch.ReleaseChannel] = branchId
				}

				// Fetch runtime versions associated *specifically* with this branch context
				runtimeVersionsForBranch, err := branchStore.GetRuntimeVersionsWithUpdateStats(ctx, app.Id, branch.BranchName)
				if err != nil {
					log.Printf("Error fetching runtime versions for branch '%s' of app '%s': %v", branch.BranchName, app.Id, err)
					return err
				}

				for _, rv := range runtimeVersionsForBranch {
					if !insertedRuntimeVersions[rv.RuntimeVersion] {
						rvParams := pgdb.MigrateLegacyRuntimeVersionParams{
							AppID:     store.ToPgUUID(app.Id),
							Version:   rv.RuntimeVersion,
							CreatedAt: parseRFC3339ToTz(rv.CreatedAt, "createdAt"),
							UpdatedAt: parseRFC3339ToTz(rv.LastUpdatedAt, "updatedAt"),
						}
						if err := qtx.MigrateLegacyRuntimeVersion(ctx, rvParams); err != nil {
							log.Printf("Error inserting runtime version '%s' for app '%s': %v", rv.RuntimeVersion, app.Id, err)
							return err
						}
						insertedRuntimeVersions[rv.RuntimeVersion] = true
					}

					updatesForRV, err := updateStore.GetUpdatesByRunTimeVersionAndBranchName(ctx, app.Id, rv.RuntimeVersion, branch.BranchName)
					if err != nil {
						log.Printf("Error fetching updates for runtime version '%s' and branch '%s' of app '%s': %v", rv.RuntimeVersion, branch.BranchName, app.Id, err)
						return err
					}

					for _, update := range updatesForRV {
						updateIdInt, err := strconv.ParseInt(update.UpdateId, 10, 64)
						if err != nil {
							log.Printf("Malformed update ID '%s': %v", update.UpdateId, err)
							return err
						}
						updateType, err := updateStore.GetUpdateType(ctx, types.Update{
							AppId:          app.Id,
							Branch:         branch.BranchName,
							RuntimeVersion: rv.RuntimeVersion,
							UpdateId:       update.UpdateId,
						})
						if err != nil {
							log.Printf("Error fetching update type for update '%s' of app '%s': %v", update.UpdateId, app.Id, err)
							return err
						}

						var messagePtr *string
						if update.Message != "" {
							localMsg := update.Message
							messagePtr = &localMsg
						}

						checkedAtTime := update2.GetUpdateCheckStatus(types.Update{
							AppId:          app.Id,
							Branch:         branch.BranchName,
							RuntimeVersion: rv.RuntimeVersion,
							UpdateId:       update.UpdateId,
						})

						updateParams := pgdb.MigrateLegacyUpdateParams{
							ID:         updateIdInt,
							AppID:      store.ToPgUUID(app.Id),
							Name:       branch.BranchName,
							Version:    rv.RuntimeVersion,
							UpdateType: int32(updateType),
							Platform:   update.Platform,
							CommitHash: update.CommitHash,
							Message:    messagePtr,
							CheckedAt:  toTimestamptz(checkedAtTime),
							UpdateUuid: store.ToPgUUID(update.UpdateUUID),
							CreatedAt:  parseRFC3339ToTz(update.CreatedAt, "createdAt"),
						}

						if err := qtx.MigrateLegacyUpdate(ctx, updateParams); err != nil {
							log.Printf("Error inserting update '%s' for app '%s': %v", update.UpdateId, app.Id, err)
							return err
						}
					}
				}
			}

			channels, err := channelStore.GetChannels(ctx, app.Id)
			if err != nil {
				log.Printf("Error fetching channels for app '%s': %v", app.Id, err)
				return err
			}

			for _, channel := range channels {
				var branchIDPtr *int64
				if id, exists := channelNameToBranchId[channel.ReleaseChannelName]; exists {
					allocatedID := id
					branchIDPtr = &allocatedID
				}

				channelParams := pgdb.MigrateLegacyChannelParams{
					AppID:    store.ToPgUUID(app.Id),
					Name:     channel.ReleaseChannelName,
					BranchID: branchIDPtr,
				}

				if err := qtx.MigrateLegacyChannel(ctx, channelParams); err != nil {
					log.Printf("Error inserting channel '%s' for app '%s': %v", channel.ReleaseChannelName, app.Id, err)
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	log.Println("🎉 [DATABASE] Successfully migrated infrastructure from env/bucket to the database.")
	return nil
}

func DownMigrateEnvJSON(ctx context.Context, tx *sql.Tx) error {
	return nil
}
