package infrastructure

import (
	"expo-open-ota/config"
	dashutils "expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/metrics"
	"expo-open-ota/internal/middleware"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func getDashboardPath() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	if strings.Contains(exePath, "/var/folders/") || strings.Contains(exePath, "Temp") {
		workingDir, _ := os.Getwd()
		return filepath.Join(workingDir, "apps", "dashboard", "dist")
	}
	return filepath.Join(exeDir, "apps", "dashboard", "dist")
}

func NewRouter(container *AppContainer) *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)

	if config.GetEnv("PROMETHEUS_ENABLED") == "true" {
		r.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			metrics.PrometheusHandler().ServeHTTP(w, r)
		}).Methods(http.MethodGet)
	}

	r.HandleFunc("/hc", HealthCheck).Methods(http.MethodGet)

	appSubrouter := r.PathPrefix("/{APP_ID}").Subrouter()
	appSubrouter.Use(middleware.AppResolverMiddleware(container.AppRepo))
	appSubrouter.HandleFunc("/requestUploadUrl/{BRANCH}", container.UploadHandler.RequestUploadUrlHandler).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/uploadLocalFile", container.UploadHandler.RequestUploadLocalFileHandler).Methods(http.MethodPut)
	appSubrouter.HandleFunc("/markUpdateAsUploaded/{BRANCH}", container.UploadHandler.MarkUpdateAsUploadedHandler).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/rollback/{BRANCH}", container.RollbackHandler.HandleRollback).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/republish/{BRANCH}", container.RepublishHandler.HandleRepublish).Methods(http.MethodPost)

	r.HandleFunc("/manifest", container.ExpoProtocolHandler.HandleManifest).Methods(http.MethodGet)
	r.HandleFunc("/assets", container.ExpoProtocolHandler.HandleAssets).Methods(http.MethodGet)

	corsSubrouter := r.PathPrefix("/auth").Subrouter()
	corsSubrouter.HandleFunc("/login", container.AuthHandler.LoginHandler).Methods(http.MethodPost)
	corsSubrouter.HandleFunc("/refreshToken", container.AuthHandler.RefreshTokenHandler).Methods(http.MethodPost)

	// Enterprise SSO (control-plane only). Pre-auth by nature: config feeds
	// the login page's SSO button, login/callback are the OIDC round-trip.
	// Registered unconditionally like the license routes; without a database,
	// a configuration or a valid license they answer accordingly.
	corsSubrouter.HandleFunc("/sso/config", container.SSOHandler.GetPublicConfigHandler).Methods(http.MethodGet)
	corsSubrouter.HandleFunc("/sso/login", container.SSOHandler.LoginRedirectHandler).Methods(http.MethodGet)
	corsSubrouter.HandleFunc("/sso/callback", container.SSOHandler.CallbackHandler).Methods(http.MethodGet)

	dashboardPath := getDashboardPath()

	if dashutils.IsDashboardEnabled() {
		r.PathPrefix("/dashboard").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get env.js
			if r.URL.Path == "/dashboard/env.js" {
				w.Header().Set("Content-Type", "application/javascript")
				baseURL := config.GetEnv("BASE_URL")
				if baseURL == "" {
					baseURL = "http://localhost:3000"
				}
				w.Write([]byte(fmt.Sprintf("window.env = { VITE_OTA_API_URL: '%s' };", baseURL)))
				return
			}
			if r.URL.Path == "/dashboard" {
				target := "/dashboard/"
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			staticExtensions := []string{".css", ".js", ".svg", ".png", ".json", ".ico"}
			for _, ext := range staticExtensions {
				if len(r.URL.Path) > len(ext) && r.URL.Path[len(r.URL.Path)-len(ext):] == ext {
					filePath := filepath.Join(dashboardPath, r.URL.Path[len("/dashboard/"):])
					if !strings.HasPrefix(filePath, dashboardPath) {
						http.Error(w, "Forbidden", http.StatusForbidden)
						return
					}
					http.ServeFile(w, r, filePath)
					return
				}
			}
			filePath := filepath.Join(dashboardPath, "index.html")
			fmt.Println("Serving file", filePath)
			http.ServeFile(w, r, filePath)
		}))
	}

	authSubrouter := r.PathPrefix("/api").Subrouter()
	authSubrouter.Use(middleware.NewAuthMiddleware(container.DashboardAuthService, container.CliAuthService))
	authSubrouter.HandleFunc("/settings", container.SettingsHandler.GetSettingsHandler).Methods(http.MethodGet)

	// adminOnly makes member accounts read-only: every dashboard mutation —
	// users, apps, branches, channels, mappings, API tokens — plus the signing
	// certificate download is admin-only. Members keep the GET routes, /api/me
	// and their own password change. It wraps individual routes rather than a
	// subrouter because admin and non-admin routes share path prefixes.
	adminOnly := middleware.NewAdminMiddleware(container.UserRepo)

	// Current account
	authSubrouter.HandleFunc("/me", container.UsersHandler.GetMeHandler).Methods(http.MethodGet)
	authSubrouter.HandleFunc("/me/password", container.UsersHandler.ChangeMyPasswordHandler).Methods(http.MethodPut)

	// Enterprise license (control-plane only). Status is readable by every
	// signed-in account so the dashboard can reflect the edition; activating
	// or removing the key is admin-only.
	authSubrouter.HandleFunc("/license", container.LicenseHandler.GetLicenseHandler).Methods(http.MethodGet)
	authSubrouter.Handle("/license", adminOnly(http.HandlerFunc(container.LicenseHandler.ActivateLicenseHandler))).Methods(http.MethodPut)
	authSubrouter.Handle("/license", adminOnly(http.HandlerFunc(container.LicenseHandler.RemoveLicenseHandler))).Methods(http.MethodDelete)

	// Enterprise SSO configuration (control-plane only, admin only), managed
	// from the dashboard's License page.
	authSubrouter.Handle("/sso", adminOnly(http.HandlerFunc(container.SSOHandler.GetConfigHandler))).Methods(http.MethodGet)
	authSubrouter.Handle("/sso", adminOnly(http.HandlerFunc(container.SSOHandler.SaveConfigHandler))).Methods(http.MethodPut)
	authSubrouter.Handle("/sso", adminOnly(http.HandlerFunc(container.SSOHandler.DeleteConfigHandler))).Methods(http.MethodDelete)

	// Users management router (control-plane only, admin only)
	authSubrouter.Handle("/users", adminOnly(http.HandlerFunc(container.UsersHandler.GetUsersHandler))).Methods(http.MethodGet)
	authSubrouter.Handle("/users", adminOnly(http.HandlerFunc(container.UsersHandler.CreateUserHandler))).Methods(http.MethodPost)
	authSubrouter.Handle("/users/{USER_ID}", adminOnly(http.HandlerFunc(container.UsersHandler.UpdateUserAdminHandler))).Methods(http.MethodPatch)
	authSubrouter.Handle("/users/{USER_ID}", adminOnly(http.HandlerFunc(container.UsersHandler.DeleteUserHandler))).Methods(http.MethodDelete)

	// Apps management router
	authSubrouter.Handle("/apps", adminOnly(http.HandlerFunc(container.AppHandler.CreateAppHandler))).Methods(http.MethodPost)
	authSubrouter.Handle("/apps/{APP_ID}", adminOnly(http.HandlerFunc(container.AppHandler.DeleteAppHandler))).Methods(http.MethodDelete)
	authSubrouter.Handle("/apps/{APP_ID}", adminOnly(http.HandlerFunc(container.AppHandler.UpdateAppHandler))).Methods(http.MethodPatch)
	authSubrouter.HandleFunc("/apps", container.AppHandler.GetAppsHandler).Methods(http.MethodGet)
	// The signing certificate is key material — admin eyes only.
	authSubrouter.Handle("/apps/{APP_ID}/certificate", adminOnly(http.HandlerFunc(container.AppHandler.DownloadAppCertificateHandler))).Methods(http.MethodGet)

	// App-scoped dashboard routes: Auth first, then AppResolver validates the
	// id and short-circuits unknown apps with 404 before handlers run. Without
	// the resolver, an unknown id falls through to bucket lookups that return
	// empty lists — the client sees 200 with [] instead of a proper "no such
	// app" signal.
	appAuthSubrouter := authSubrouter.PathPrefix("/apps/{APP_ID}").Subrouter()
	appAuthSubrouter.StrictSlash(true)
	appAuthSubrouter.Use(middleware.AppResolverMiddleware(container.AppRepo))
	appAuthSubrouter.HandleFunc("/", container.AppHandler.GetAppHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/branches", adminOnly(http.HandlerFunc(container.BranchHandler.CreateBranchHandler))).Methods(http.MethodPost)
	appAuthSubrouter.Handle("/branches/{BRANCH}", adminOnly(http.HandlerFunc(container.BranchHandler.DeleteBranchHandler))).Methods(http.MethodDelete)
	appAuthSubrouter.HandleFunc("/branches", container.BranchHandler.GetBranchesHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/channels", adminOnly(http.HandlerFunc(container.ChannelHandler.CreateChannelHandler))).Methods(http.MethodPost)
	appAuthSubrouter.Handle("/channels/{CHANNEL}", adminOnly(http.HandlerFunc(container.ChannelHandler.DeleteChannelHandler))).Methods(http.MethodDelete)
	appAuthSubrouter.HandleFunc("/channels", container.ChannelHandler.GetChannelsHandler).Methods(http.MethodGet)
	// Progressive rollouts (control-plane only; reads stay open like the sibling
	// listings, every mutation is admin-only). Channel rollouts are keyed by channel
	// name, per-update rollouts by (branch, runtime version).
	appAuthSubrouter.HandleFunc("/channels/{CHANNEL}/rollout", container.RolloutHandler.GetChannelRolloutHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/channels/{CHANNEL}/rollout", adminOnly(http.HandlerFunc(container.RolloutHandler.StartChannelRolloutHandler))).Methods(http.MethodPost)
	appAuthSubrouter.Handle("/channels/{CHANNEL}/rollout", adminOnly(http.HandlerFunc(container.RolloutHandler.UpdateChannelRolloutHandler))).Methods(http.MethodPatch)
	appAuthSubrouter.Handle("/channels/{CHANNEL}/rollout/end", adminOnly(http.HandlerFunc(container.RolloutHandler.EndChannelRolloutHandler))).Methods(http.MethodPost)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/rollout", container.RolloutHandler.GetUpdateRolloutHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/rollout", adminOnly(http.HandlerFunc(container.RolloutHandler.SetUpdateRolloutPercentageHandler))).Methods(http.MethodPut)
	appAuthSubrouter.Handle("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/rollout/revert", adminOnly(http.HandlerFunc(container.RolloutHandler.RevertUpdateRolloutHandler))).Methods(http.MethodPost)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersions", container.BranchHandler.GetRuntimeVersionsHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/updates", container.UpdateHandler.GetUpdatesHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/updates/{UPDATE_ID}", container.UpdateHandler.GetUpdateDetailsHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/branch/{BRANCH_ID}/updateChannelBranchMapping", adminOnly(http.HandlerFunc(container.BranchHandler.UpdateChannelBranchMappingHandler))).Methods(http.MethodPost)
	// An API token is publishing power over the app — minting and revoking are
	// admin actions. The list stays readable: it only carries names and hints.
	appAuthSubrouter.Handle("/apiKeys", adminOnly(http.HandlerFunc(container.ApiKeyHandler.CreateApiKeyHandler))).Methods(http.MethodPost)
	appAuthSubrouter.HandleFunc("/apiKeys", container.ApiKeyHandler.GetApiKeysHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/apiKeys/{API_KEY_ID}/revoke", adminOnly(http.HandlerFunc(container.ApiKeyHandler.RevokeApiKeyHandler))).Methods(http.MethodDelete)
	// Enterprise: per-key access restrictions (protected-branch access + IP
	// allowlist) and branch protection. Reads stay open like the key list;
	// the writes change what a token can do, so they are admin-only and
	// license-gated in the service.
	appAuthSubrouter.HandleFunc("/apiKeys/restrictions", container.ApiKeyRestrictionHandler.GetApiKeyRestrictionsHandler).Methods(http.MethodGet)
	appAuthSubrouter.Handle("/apiKeys/{API_KEY_ID}/restrictions", adminOnly(http.HandlerFunc(container.ApiKeyRestrictionHandler.SetApiKeyRestrictionsHandler))).Methods(http.MethodPut)
	appAuthSubrouter.Handle("/branches/{BRANCH}/protection", adminOnly(http.HandlerFunc(container.ApiKeyRestrictionHandler.SetBranchProtectionHandler))).Methods(http.MethodPut)
	return r
}
