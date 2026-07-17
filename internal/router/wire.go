package infrastructure

import (
	"context"
	"expo-open-ota/config"
	"expo-open-ota/internal/bucket"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres"
	"expo-open-ota/internal/database/postgres/migrations"
	"expo-open-ota/internal/handlers"
	dashhandlers "expo-open-ota/internal/handlers/dashboard"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/store"
	"log"
)

type AppContainer struct {
	AuthHandler          *dashhandlers.AuthHandler
	DashboardAuthService *services.DashboardAuthService
	CliAuthService       *services.CliAuthService
	ApiKeyHandler        *dashhandlers.ApiKeyHandler
	AppHandler           *dashhandlers.AppHandler
	AppRepo              services.AppRepository
	BranchHandler        *dashhandlers.BranchHandler
	ChannelHandler       *dashhandlers.ChannelHandler
	ExpoProtocolHandler  *handlers.ExpoProtocolHandler
	RollbackHandler      *handlers.RollbackHandler
	SettingsHandler      *dashhandlers.SettingsHandler
	UpdateHandler        *dashhandlers.UpdateHandler
	UploadHandler        *handlers.UploadHandler
	RepublishHandler     *handlers.RepublishHandler
}

// logLegacyAppIdFallback states, once at boot, which app receives manifest and
// asset requests that carry no expo-app-id header. Whether v1 clients still get
// updates hinges on this and is otherwise invisible until someone notices an
// install has silently stopped updating, so it is worth a line in the log.
func logLegacyAppIdFallback() {
	if appId := config.LegacyFallbackAppId(); appId != "" {
		log.Printf("🔁 [LEGACY] app id fallback ACTIVE for %s — v1 clients sending no expo-app-id header resolve to this app. Set SKIP_LEGACY_APP_ID_FALLBACK=true once every client ships the header.", appId)
		return
	}
	if config.GetEnv("EXPO_APP_ID") != "" {
		log.Println("🔒 [LEGACY] app id fallback DISABLED by SKIP_LEGACY_APP_ID_FALLBACK — manifest/asset requests without an expo-app-id header are rejected. Any v1 client that has not been rebuilt stops receiving updates.")
	}
}

func InitDependencies(ctx context.Context) (*AppContainer, func()) {
	var authRepo services.CliAuthRepository
	var appRepo services.AppRepository
	var branchRepo services.BranchRepository
	var channelRepo services.ChannelRepository
	var updateRepo services.UpdateRepository

	cleanup := func() {}
	dbUrl := config.GetDBURL()

	resolvedBucket := bucket.GetBucket()

	if dbUrl != "" {
		if !database.IsValidDBURL(dbUrl) {
			log.Fatalf("Invalid database URL: %s", dbUrl)
		}
		err := config.ValidateMasterKey()
		if err != nil {
			log.Fatalf("Invalid master key configuration: %v", err)
		}
		log.Println("⚙️  [CONTROL] Initializing Control Plane (DB Mode)..")
		dbConfig := database.LoadDBConfigFromEnv()
		dbEngine, err := database.NewPostgresEngine(ctx, dbConfig)
		if err != nil {
			log.Fatalf("Database initialization failed: %v", err)
		}
		cleanup = func() { dbEngine.Close() }
		migrations.SetEngine(dbEngine)
		postgres.RunDBMigrations(dbUrl)

		authRepo = store.NewPostgresAuthStore(dbEngine)
		appRepo = store.NewPostgresAppStore(dbEngine)
		branchRepo = store.NewPostgresBranchStore(dbEngine)
		channelRepo = store.NewPostgresChannelStore(dbEngine)
		updateRepo = store.NewPostgresUpdateStore(dbEngine)
	} else {
		log.Println("⚙️  [STATELESS] Initializing Stateless Mode (Flat-Env Mode)...")
		if err := config.LoadAppsFromFlatEnv(); err != nil {
			log.Fatalf("Invalid apps config: %v\nSee https://mercure-technologies.gitbook.io/expo-open-ota/stateless-mode/getting-started for the stateless (flat-env) config format.", err)
		}
		authRepo = store.NewBucketAuthStore(resolvedBucket)
		appRepo = store.NewBucketAppStore(resolvedBucket)
		branchRepo = store.NewBucketBranchStore(resolvedBucket)
		channelRepo = store.NewBucketChannelStore(resolvedBucket)
		updateRepo = store.NewBucketUpdateStore(resolvedBucket)
	}

	logLegacyAppIdFallback()

	dashboardAuthService := services.NewDashboardAuthService()
	cliAuthService := services.NewCliAuthService(authRepo)
	appService := services.NewAppService(appRepo)
	branchService := services.NewBranchService(branchRepo, channelRepo, updateRepo, resolvedBucket)
	channelService := services.NewChannelService(branchRepo, channelRepo)
	updateService := services.NewUpdateService(updateRepo, resolvedBucket)
	expoProtocolService := services.NewExpoProtocolService(appRepo, channelRepo, updateRepo, updateService)
	deploymentService := services.NewDeploymentService(branchService, updateService, updateRepo, resolvedBucket)

	return &AppContainer{
		AuthHandler:          dashhandlers.NewAuthHandler(dashboardAuthService),
		DashboardAuthService: dashboardAuthService,
		CliAuthService:       cliAuthService,
		ApiKeyHandler:        dashhandlers.NewApiKeyHandler(cliAuthService),
		AppHandler:           dashhandlers.NewAppHandler(appService),
		AppRepo:              appRepo,
		BranchHandler:        dashhandlers.NewBranchHandler(branchService),
		ChannelHandler:       dashhandlers.NewChannelHandler(channelService),
		ExpoProtocolHandler:  handlers.NewExpoProtocolHandler(expoProtocolService),
		RepublishHandler:     handlers.NewRepublishHandler(cliAuthService, deploymentService),
		RollbackHandler:      handlers.NewRollbackHandler(cliAuthService, deploymentService),
		SettingsHandler:      dashhandlers.NewSettingsHandler(appService),
		UpdateHandler:        dashhandlers.NewUpdateHandler(updateService),
		UploadHandler:        handlers.NewUploadHandler(cliAuthService, deploymentService),
	}, cleanup
}
