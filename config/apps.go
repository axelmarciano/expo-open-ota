package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// KeysMode identifies how an app's signing key pair is stored.
type KeysMode string

const (
	KeysModeLocal       KeysMode = "local"
	KeysModeAWSSM       KeysMode = "aws-secrets-manager"
	KeysModeEnvironment KeysMode = "environment"
)

// KeysConfig is a tagged union: exactly one set of fields must be populated,
// matched by Mode. Validated at LoadConfig time — downstream code assumes a
// valid config and does not re-check invariants.
type KeysConfig struct {
	Mode KeysMode `json:"mode"`

	// mode=local
	PublicPath  string `json:"publicPath,omitempty"`
	PrivatePath string `json:"privatePath,omitempty"`

	// mode=aws-secrets-manager
	PublicSecretId  string `json:"publicSecretId,omitempty"`
	PrivateSecretId string `json:"privateSecretId,omitempty"`

	// mode=environment
	PublicB64  string `json:"publicB64,omitempty"`
	PrivateB64 string `json:"privateB64,omitempty"`
}

// AppConfig is one entry of the EXPO_APPS_JSON config. Each app has its own
// identity (id, accessToken) and signing key pair. Name is optional and used
// purely as a display label in the dashboard — it does not participate in
// request routing, which always goes by Id.
type AppConfig struct {
	Id          string     `json:"id"`
	Name        string     `json:"name,omitempty"`
	AccessToken string     `json:"accessToken"`
	Keys        KeysConfig `json:"keys"`
}

// AppDescriptor is the public-safe view of an AppConfig (no token, no keys)
// intended for dashboard listings and anything else that needs to enumerate
// apps without touching secrets.
type AppDescriptor struct {
	Id   string `json:"id"`
	Name string `json:"name,omitempty"`
}

var (
	appsByIdMu sync.RWMutex
	appsById   map[string]*AppConfig
)

// LoadApps resolves the multi-app config from env, validates it, and caches
// the result in memory. Two sources are supported, in priority order:
//
//  1. EXPO_APPS_JSON — a JSON array of AppConfig entries, used for multi-app
//     deployments. This is the "multi-app" path.
//  2. Flat env vars (EXPO_APP_ID, EXPO_ACCESS_TOKEN, KEYS_STORAGE_TYPE and its
//     mode-specific siblings) — parsed into a single-element array. This is
//     the "simple 1-app" path and mirrors the v1 env layout unchanged, so a
//     v1 install upgrades to v2 with zero config changes.
//
// Must be called once from LoadConfig before any handler resolves an app.
// Returns a non-nil error on any structural or semantic issue; callers are
// expected to log.Fatal on error.
func LoadApps() error {
	apps, source, err := readApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		return fmt.Errorf("%s: must declare at least one app", source)
	}
	index := make(map[string]*AppConfig, len(apps))
	for i := range apps {
		app := &apps[i]
		if err := validateApp(app, i); err != nil {
			return fmt.Errorf("%s: %w", source, err)
		}
		if _, dup := index[app.Id]; dup {
			return fmt.Errorf("%s: duplicate app id %q", source, app.Id)
		}
		index[app.Id] = app
	}
	appsByIdMu.Lock()
	appsById = index
	appsByIdMu.Unlock()
	return nil
}

// readApps returns the parsed (but not yet validated) list of apps plus a
// human-readable source tag used for error messages. EXPO_APPS_JSON wins when
// set. The flat-env fallback reads legacy v1 variable names verbatim to
// preserve upgrade-in-place.
func readApps() ([]AppConfig, string, error) {
	if inline := strings.TrimSpace(os.Getenv("EXPO_APPS_JSON")); inline != "" {
		var apps []AppConfig
		if err := json.Unmarshal([]byte(inline), &apps); err != nil {
			return nil, "EXPO_APPS_JSON", fmt.Errorf("EXPO_APPS_JSON: invalid JSON: %w", err)
		}
		return apps, "EXPO_APPS_JSON", nil
	}
	if appId := strings.TrimSpace(os.Getenv("EXPO_APP_ID")); appId != "" {
		return []AppConfig{loadFromFlatEnv(appId)}, "flat env (EXPO_APP_ID)", nil
	}
	return nil, "", fmt.Errorf("no apps config found: set EXPO_APPS_JSON for multi-app, or EXPO_APP_ID + EXPO_ACCESS_TOKEN + key vars for the single-app case")
}

// loadFromFlatEnv reads the v1 single-app env vars and returns an AppConfig.
// Validation is deferred to validateApp so the error path is uniform with the
// JSON case.
func loadFromFlatEnv(appId string) AppConfig {
	app := AppConfig{
		Id:          appId,
		AccessToken: os.Getenv("EXPO_ACCESS_TOKEN"),
	}
	switch os.Getenv("KEYS_STORAGE_TYPE") {
	case "local", "":
		// "local" is the default when unset, matching v1 DefaultEnvValues.
		app.Keys = KeysConfig{
			Mode:        KeysModeLocal,
			PublicPath:  os.Getenv("PUBLIC_LOCAL_EXPO_KEY_PATH"),
			PrivatePath: os.Getenv("PRIVATE_LOCAL_EXPO_KEY_PATH"),
		}
	case "aws-secrets-manager":
		app.Keys = KeysConfig{
			Mode:            KeysModeAWSSM,
			PublicSecretId:  os.Getenv("AWSSM_EXPO_PUBLIC_KEY_SECRET_ID"),
			PrivateSecretId: os.Getenv("AWSSM_EXPO_PRIVATE_KEY_SECRET_ID"),
		}
	case "environment":
		app.Keys = KeysConfig{
			Mode:       KeysModeEnvironment,
			PublicB64:  os.Getenv("PUBLIC_EXPO_KEY_B64"),
			PrivateB64: os.Getenv("PRIVATE_EXPO_KEY_B64"),
		}
	default:
		// Leave Mode empty; validateApp will surface a clear error naming
		// the invalid KEYS_STORAGE_TYPE in the boot log.
	}
	return app
}

func validateApp(app *AppConfig, index int) error {
	prefix := fmt.Sprintf("apps[%d]", index)
	if app.Id == "" {
		return fmt.Errorf("%s.id is required", prefix)
	}
	if strings.ContainsAny(app.Id, "/\\ \t\n") {
		return fmt.Errorf("%s.id %q must not contain whitespace or path separators", prefix, app.Id)
	}
	// Reserved filesystem names — match validateSegment / isValidAppID so
	// all three id-validation paths agree. An app id of "." or ".." would
	// resolve to the bucket root (or its parent) when interpolated into
	// {appId}/{branch}/… on the local backend.
	if app.Id == "." || app.Id == ".." {
		return fmt.Errorf("%s.id %q is reserved", prefix, app.Id)
	}
	if app.AccessToken == "" {
		return fmt.Errorf("%s.accessToken is required", prefix)
	}
	return validateKeys(&app.Keys, prefix+".keys")
}

func validateKeys(k *KeysConfig, prefix string) error {
	switch k.Mode {
	case KeysModeLocal:
		if k.PublicPath == "" || k.PrivatePath == "" {
			return fmt.Errorf("%s: mode=local requires publicPath and privatePath", prefix)
		}
		if k.PublicSecretId != "" || k.PrivateSecretId != "" || k.PublicB64 != "" || k.PrivateB64 != "" {
			return fmt.Errorf("%s: mode=local must not set aws-sm or b64 fields", prefix)
		}
	case KeysModeAWSSM:
		if k.PublicSecretId == "" || k.PrivateSecretId == "" {
			return fmt.Errorf("%s: mode=aws-secrets-manager requires publicSecretId and privateSecretId", prefix)
		}
		if k.PublicPath != "" || k.PrivatePath != "" || k.PublicB64 != "" || k.PrivateB64 != "" {
			return fmt.Errorf("%s: mode=aws-secrets-manager must not set local or b64 fields", prefix)
		}
	case KeysModeEnvironment:
		if k.PublicB64 == "" || k.PrivateB64 == "" {
			return fmt.Errorf("%s: mode=environment requires publicB64 and privateB64", prefix)
		}
		if k.PublicPath != "" || k.PrivatePath != "" || k.PublicSecretId != "" || k.PrivateSecretId != "" {
			return fmt.Errorf("%s: mode=environment must not set local or aws-sm fields", prefix)
		}
	case "":
		return fmt.Errorf("%s.mode is required (local|aws-secrets-manager|environment)", prefix)
	default:
		return fmt.Errorf("%s.mode=%q is invalid (expected local|aws-secrets-manager|environment)", prefix, k.Mode)
	}
	return nil
}

// ListAppIds returns the configured app ids in an unspecified order. Meant
// for read-only introspection (e.g. the dashboard /api/settings endpoint);
// never expose tokens or keys from the same path.
func ListAppIds() []string {
	appsByIdMu.RLock()
	defer appsByIdMu.RUnlock()
	ids := make([]string, 0, len(appsById))
	for id := range appsById {
		ids = append(ids, id)
	}
	return ids
}

// ListApps returns a public-safe descriptor for each configured app. Same
// contract as ListAppIds (unspecified order, snapshot-at-call-time) but
// includes the optional display name so clients like the dashboard can show
// a human-readable label instead of the raw UUID.
func ListApps() []AppDescriptor {
	appsByIdMu.RLock()
	defer appsByIdMu.RUnlock()
	out := make([]AppDescriptor, 0, len(appsById))
	for _, app := range appsById {
		out = append(out, AppDescriptor{Id: app.Id, Name: app.Name})
	}
	return out
}

// GetAppConfig returns the resolved configuration for the given app id.
// Returns an error when the id is unknown so callers can return a clear 404
// instead of silently serving a different app's content.
func GetAppConfig(appId string) (*AppConfig, error) {
	appsByIdMu.RLock()
	defer appsByIdMu.RUnlock()
	app, ok := appsById[appId]
	if !ok {
		return nil, fmt.Errorf("unknown app id %q", appId)
	}
	return app, nil
}

// ResetAppsForTest clears the loaded apps cache. Intended for tests that
// need to reinitialize the config between runs. Unexported-equivalent
// naming keeps it out of "normal" autocompletion.
func ResetAppsForTest() {
	appsByIdMu.Lock()
	appsById = nil
	appsByIdMu.Unlock()
}
