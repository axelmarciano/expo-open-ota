package infrastructure

import (
	"expo-open-ota/config"
	"expo-open-ota/internal/dashboard"
	"expo-open-ota/internal/handlers"
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

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.PrometheusHandler().ServeHTTP(w, r)
	}).Methods(http.MethodGet)

	r.HandleFunc("/hc", HealthCheck).Methods(http.MethodGet)


	appSubrouter := r.PathPrefix("/{APP_ID}").Subrouter()
	appSubrouter.Use(middleware.AppResolverMiddleware)
	appSubrouter.HandleFunc("/requestUploadUrl/{BRANCH}", handlers.RequestUploadUrlHandler).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/uploadLocalFile", handlers.RequestUploadLocalFileHandler).Methods(http.MethodPut)
	appSubrouter.HandleFunc("/markUpdateAsUploaded/{BRANCH}", handlers.MarkUpdateAsUploadedHandler).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/rollback/{BRANCH}", handlers.RollbackHandler).Methods(http.MethodPost)
	appSubrouter.HandleFunc("/republish/{BRANCH}", handlers.RepublishHandler).Methods(http.MethodPost)

	r.HandleFunc("/manifest", handlers.ManifestHandler).Methods(http.MethodGet)
	r.HandleFunc("/assets", handlers.AssetsHandler).Methods(http.MethodGet)

	corsSubrouter := r.PathPrefix("/auth").Subrouter()
	corsSubrouter.HandleFunc("/login", handlers.LoginHandler).Methods(http.MethodPost)
	corsSubrouter.HandleFunc("/refreshToken", handlers.RefreshTokenHandler).Methods(http.MethodPost)

	dashboardPath := getDashboardPath()

	if dashboard.IsDashboardEnabled() {
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
	authSubrouter.Use(middleware.AuthMiddleware)
	authSubrouter.HandleFunc("/settings", handlers.GetSettingsHandler).Methods(http.MethodGet)

	// App-scoped dashboard routes: Auth first, then AppResolver validates the
	// id and short-circuits unknown apps with 404 before handlers run. Without
	// the resolver, an unknown id falls through to bucket lookups that return
	// empty lists — the client sees 200 with [] instead of a proper "no such
	// app" signal.
	appAuthSubrouter := authSubrouter.PathPrefix("/apps/{APP_ID}").Subrouter()
	appAuthSubrouter.Use(middleware.AppResolverMiddleware)
	appAuthSubrouter.HandleFunc("/branches", handlers.GetBranchesHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/channels", handlers.GetChannelsHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersions", handlers.GetRuntimeVersionsHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/updates", handlers.GetUpdatesHandler).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/runtimeVersion/{RUNTIME_VERSION}/updates/{UPDATE_ID}", handlers.GetUpdateDetails).Methods(http.MethodGet)
	appAuthSubrouter.HandleFunc("/branch/{BRANCH}/updateChannelBranchMapping", handlers.UpdateChannelBranchMappingHandler).Methods(http.MethodPost)
	return r
}
