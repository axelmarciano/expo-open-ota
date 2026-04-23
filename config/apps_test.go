package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetAppsEnv unsets every env var that LoadApps can read so each test
// starts from a known-empty environment. Central list — when a new env var
// is added to the loader, add it here too or the test suite will start
// interfering across runs.
func resetAppsEnv(t *testing.T) {
	t.Helper()
	vars := []string{
		"EXPO_APPS_JSON",
		"EXPO_APP_ID",
		"EXPO_ACCESS_TOKEN",
		"KEYS_STORAGE_TYPE",
		"PUBLIC_LOCAL_EXPO_KEY_PATH",
		"PRIVATE_LOCAL_EXPO_KEY_PATH",
		"AWSSM_EXPO_PUBLIC_KEY_SECRET_ID",
		"AWSSM_EXPO_PRIVATE_KEY_SECRET_ID",
		"PUBLIC_EXPO_KEY_B64",
		"PRIVATE_EXPO_KEY_B64",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
	ResetAppsForTest()
	t.Cleanup(func() {
		for _, v := range vars {
			os.Unsetenv(v)
		}
		ResetAppsForTest()
	})
}

// -----------------------------------------------------------------------------
// validateApp / validateKeys — unit tests on the validator only. No env.
// -----------------------------------------------------------------------------

func TestValidateApp_AcceptsEachMode(t *testing.T) {
	cases := map[string]AppConfig{
		"local": {
			Id:          "app-1",
			AccessToken: "token",
			Keys: KeysConfig{
				Mode:        KeysModeLocal,
				PublicPath:  "/keys/pub.pem",
				PrivatePath: "/keys/priv.pem",
			},
		},
		"aws-secrets-manager": {
			Id:          "app-1",
			AccessToken: "token",
			Keys: KeysConfig{
				Mode:            KeysModeAWSSM,
				PublicSecretId:  "/eoota/pub",
				PrivateSecretId: "/eoota/priv",
			},
		},
		"environment": {
			Id:          "app-1",
			AccessToken: "token",
			Keys: KeysConfig{
				Mode:       KeysModeEnvironment,
				PublicB64:  "cHViLXBlbQ==",
				PrivateB64: "cHJpdi1wZW0=",
			},
		},
	}
	for name, app := range cases {
		t.Run(name, func(t *testing.T) {
			assert.NoError(t, validateApp(&app, 0))
		})
	}
}

func TestValidateApp_RejectsBadId(t *testing.T) {
	cases := map[string]string{
		"empty":     "",
		"slash":     "foo/bar",
		"backslash": "foo\\bar",
		"space":     "foo bar",
		"tab":       "foo\tbar",
		"newline":   "foo\nbar",
		"dot":       ".",
		"dotdot":    "..",
	}
	for name, badId := range cases {
		t.Run(name, func(t *testing.T) {
			app := AppConfig{
				Id:          badId,
				AccessToken: "token",
				Keys: KeysConfig{
					Mode:        KeysModeLocal,
					PublicPath:  "/p",
					PrivatePath: "/q",
				},
			}
			err := validateApp(&app, 0)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "apps[0].id")
		})
	}
}

func TestValidateApp_RejectsMissingAccessToken(t *testing.T) {
	app := AppConfig{
		Id:          "app-1",
		AccessToken: "",
		Keys: KeysConfig{
			Mode:        KeysModeLocal,
			PublicPath:  "/p",
			PrivatePath: "/q",
		},
	}
	err := validateApp(&app, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apps[2].accessToken")
}

func TestValidateKeys_RejectsMissingModeFields(t *testing.T) {
	cases := map[string]KeysConfig{
		"local missing public":  {Mode: KeysModeLocal, PrivatePath: "/q"},
		"local missing private": {Mode: KeysModeLocal, PublicPath: "/p"},
		"aws-sm missing public": {Mode: KeysModeAWSSM, PrivateSecretId: "/q"},
		"aws-sm missing private": {Mode: KeysModeAWSSM, PublicSecretId: "/p"},
		"environment missing public":  {Mode: KeysModeEnvironment, PrivateB64: "xx"},
		"environment missing private": {Mode: KeysModeEnvironment, PublicB64: "xx"},
	}
	for name, k := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateKeys(&k, "apps[0].keys")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "requires")
		})
	}
}

func TestValidateKeys_RejectsCrossModeFields(t *testing.T) {
	cases := map[string]KeysConfig{
		"local with aws-sm field": {
			Mode: KeysModeLocal, PublicPath: "/p", PrivatePath: "/q",
			PublicSecretId: "leaked",
		},
		"local with b64 field": {
			Mode: KeysModeLocal, PublicPath: "/p", PrivatePath: "/q",
			PublicB64: "leaked",
		},
		"aws-sm with local field": {
			Mode: KeysModeAWSSM, PublicSecretId: "/p", PrivateSecretId: "/q",
			PublicPath: "leaked",
		},
		"aws-sm with b64 field": {
			Mode: KeysModeAWSSM, PublicSecretId: "/p", PrivateSecretId: "/q",
			PrivateB64: "leaked",
		},
		"environment with local field": {
			Mode: KeysModeEnvironment, PublicB64: "xx", PrivateB64: "yy",
			PublicPath: "leaked",
		},
		"environment with aws-sm field": {
			Mode: KeysModeEnvironment, PublicB64: "xx", PrivateB64: "yy",
			PrivateSecretId: "leaked",
		},
	}
	for name, k := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateKeys(&k, "apps[0].keys")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "must not set")
		})
	}
}

func TestValidateKeys_RejectsMissingOrUnknownMode(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		err := validateKeys(&KeysConfig{}, "apps[0].keys")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
	t.Run("unknown", func(t *testing.T) {
		err := validateKeys(&KeysConfig{Mode: "env-b64"}, "apps[0].keys")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
		// The "env-b64" value is a common migration mistake — the message
		// must spell out the three valid modes so the user knows to swap.
		assert.Contains(t, err.Error(), "environment")
	})
}

// -----------------------------------------------------------------------------
// LoadApps — JSON source. Covers happy path, parse failures, and the full
// validation surface when the loader drives it via env.
// -----------------------------------------------------------------------------

func TestLoadApps_FromJSON_Happy(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"a","accessToken":"ta","keys":{"mode":"local","publicPath":"/a-pub","privatePath":"/a-priv"}},
      {"id":"b","accessToken":"tb","keys":{"mode":"aws-secrets-manager","publicSecretId":"/b-pub","privateSecretId":"/b-priv"}},
      {"id":"c","accessToken":"tc","keys":{"mode":"environment","publicB64":"pub","privateB64":"priv"}}
    ]`)

	require.NoError(t, LoadApps())

	ids := ListAppIds()
	assert.ElementsMatch(t, []string{"a", "b", "c"}, ids)

	a, err := GetAppConfig("a")
	require.NoError(t, err)
	assert.Equal(t, KeysModeLocal, a.Keys.Mode)
	assert.Equal(t, "ta", a.AccessToken)

	c, err := GetAppConfig("c")
	require.NoError(t, err)
	assert.Equal(t, KeysModeEnvironment, c.Keys.Mode)
	assert.Equal(t, "pub", c.Keys.PublicB64)
}

func TestLoadApps_FromJSON_RejectsMalformed(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `not-json`)
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXPO_APPS_JSON")
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestLoadApps_FromJSON_RejectsEmptyArray(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[]`)
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one app")
}

func TestLoadApps_FromJSON_RejectsDuplicateIds(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"dup","accessToken":"ta","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"dup","accessToken":"tb","keys":{"mode":"local","publicPath":"/x","privatePath":"/y"}}
    ]`)
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate app id")
	assert.Contains(t, err.Error(), `"dup"`)
}

func TestLoadApps_FromJSON_SurfacesFieldPathInError(t *testing.T) {
	resetAppsEnv(t)
	// Second entry has the problem; the error must point at apps[1] so the
	// user doesn't have to bisect their config.
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"ok","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"broken","accessToken":"t","keys":{"mode":"local"}}
    ]`)
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apps[1].keys")
	assert.Contains(t, err.Error(), "EXPO_APPS_JSON")
}

func TestLoadApps_FromJSON_WhitespaceOnlyIsTreatedAsUnset(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", "   \n  ")
	// Whitespace EXPO_APPS_JSON must fall through — otherwise a
	// well-meaning user setting the var to "" (which can produce
	// whitespace on some platforms) gets a JSON parse error instead of
	// the friendly "no config" message.
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no apps config found")
}

// -----------------------------------------------------------------------------
// LoadApps — flat env fallback. One-app path, v1-compat. Each mode, and
// failure modes.
// -----------------------------------------------------------------------------

func TestLoadApps_FromFlatEnv_LocalMode(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	os.Setenv("EXPO_ACCESS_TOKEN", "tok")
	os.Setenv("KEYS_STORAGE_TYPE", "local")
	os.Setenv("PUBLIC_LOCAL_EXPO_KEY_PATH", "/k/pub.pem")
	os.Setenv("PRIVATE_LOCAL_EXPO_KEY_PATH", "/k/priv.pem")

	require.NoError(t, LoadApps())
	a, err := GetAppConfig("solo")
	require.NoError(t, err)
	assert.Equal(t, KeysModeLocal, a.Keys.Mode)
	assert.Equal(t, "/k/pub.pem", a.Keys.PublicPath)
	assert.Equal(t, "/k/priv.pem", a.Keys.PrivatePath)
	assert.Equal(t, "tok", a.AccessToken)
}

func TestLoadApps_FromFlatEnv_DefaultsToLocalWhenStorageTypeUnset(t *testing.T) {
	// v1 DefaultEnvValues had KEYS_STORAGE_TYPE=local; in v2 that default
	// moved into loadFromFlatEnv itself. Unsetting the var must behave the
	// same as setting it to "local".
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	os.Setenv("EXPO_ACCESS_TOKEN", "tok")
	os.Setenv("PUBLIC_LOCAL_EXPO_KEY_PATH", "/k/pub.pem")
	os.Setenv("PRIVATE_LOCAL_EXPO_KEY_PATH", "/k/priv.pem")

	require.NoError(t, LoadApps())
	a, err := GetAppConfig("solo")
	require.NoError(t, err)
	assert.Equal(t, KeysModeLocal, a.Keys.Mode)
}

func TestLoadApps_FromFlatEnv_AWSSMMode(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	os.Setenv("EXPO_ACCESS_TOKEN", "tok")
	os.Setenv("KEYS_STORAGE_TYPE", "aws-secrets-manager")
	os.Setenv("AWSSM_EXPO_PUBLIC_KEY_SECRET_ID", "/eoota/pub")
	os.Setenv("AWSSM_EXPO_PRIVATE_KEY_SECRET_ID", "/eoota/priv")

	require.NoError(t, LoadApps())
	a, err := GetAppConfig("solo")
	require.NoError(t, err)
	assert.Equal(t, KeysModeAWSSM, a.Keys.Mode)
	assert.Equal(t, "/eoota/pub", a.Keys.PublicSecretId)
	assert.Equal(t, "/eoota/priv", a.Keys.PrivateSecretId)
}

func TestLoadApps_FromFlatEnv_EnvironmentMode(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	os.Setenv("EXPO_ACCESS_TOKEN", "tok")
	os.Setenv("KEYS_STORAGE_TYPE", "environment")
	os.Setenv("PUBLIC_EXPO_KEY_B64", "cHViLWI2NA==")
	os.Setenv("PRIVATE_EXPO_KEY_B64", "cHJpdi1iNjQ=")

	require.NoError(t, LoadApps())
	a, err := GetAppConfig("solo")
	require.NoError(t, err)
	assert.Equal(t, KeysModeEnvironment, a.Keys.Mode)
	assert.Equal(t, "cHViLWI2NA==", a.Keys.PublicB64)
}

func TestLoadApps_FromFlatEnv_RejectsUnknownStorageType(t *testing.T) {
	// An unknown KEYS_STORAGE_TYPE leaves Mode empty in loadFromFlatEnv,
	// then validateApp catches it with the "mode is required" error.
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	os.Setenv("EXPO_ACCESS_TOKEN", "tok")
	os.Setenv("KEYS_STORAGE_TYPE", "vault") // not supported
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mode is required")
}

func TestLoadApps_FromFlatEnv_RejectsMissingToken(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APP_ID", "solo")
	// No EXPO_ACCESS_TOKEN, no keys set either
	err := LoadApps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accessToken")
}

// -----------------------------------------------------------------------------
// LoadApps — priority and "nothing set" error path.
// -----------------------------------------------------------------------------

func TestLoadApps_JSONWinsOverFlatEnv(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[{"id":"from-json","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}]`)
	os.Setenv("EXPO_APP_ID", "from-flat")
	os.Setenv("EXPO_ACCESS_TOKEN", "t")
	os.Setenv("PUBLIC_LOCAL_EXPO_KEY_PATH", "/p")
	os.Setenv("PRIVATE_LOCAL_EXPO_KEY_PATH", "/q")

	require.NoError(t, LoadApps())
	// The flat env's "from-flat" must not leak into the app registry —
	// sources never merge.
	assert.Equal(t, []string{"from-json"}, ListAppIds())
	_, err := GetAppConfig("from-flat")
	assert.Error(t, err)
}

func TestLoadApps_NoSourceSetReturnsActionableError(t *testing.T) {
	resetAppsEnv(t)
	err := LoadApps()
	require.Error(t, err)
	// The error message is part of the UX — it must name both paths so a
	// user who forgot to set anything isn't left guessing.
	msg := err.Error()
	assert.Contains(t, msg, "EXPO_APPS_JSON")
	assert.Contains(t, msg, "EXPO_APP_ID")
}

// -----------------------------------------------------------------------------
// GetAppConfig / ListAppIds — lookup API.
// -----------------------------------------------------------------------------

func TestGetAppConfig_UnknownIdReturnsError(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[{"id":"known","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}]`)
	require.NoError(t, LoadApps())

	_, err := GetAppConfig("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"unknown"`)
}

func TestListAppIds_EmptyWhenNoConfigLoaded(t *testing.T) {
	resetAppsEnv(t) // also resets the cache
	assert.Empty(t, ListAppIds())
}

func TestListAppIds_ReturnsAllLoaded(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"a","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"b","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"c","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}
    ]`)
	require.NoError(t, LoadApps())
	assert.ElementsMatch(t, []string{"a", "b", "c"}, ListAppIds())
}

// -----------------------------------------------------------------------------
// Concurrency — GetAppConfig / ListAppIds must be safe for concurrent reads
// (the server calls them from every handler goroutine).
// -----------------------------------------------------------------------------

func TestLookupIsConcurrencySafe(t *testing.T) {
	resetAppsEnv(t)
	// Build a decent-sized config so the map iteration in ListAppIds has
	// something to do under contention.
	var entries []string
	for i := 0; i < 50; i++ {
		entries = append(entries, fmt.Sprintf(
			`{"id":"app-%d","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}`,
			i,
		))
	}
	os.Setenv("EXPO_APPS_JSON", "["+strings.Join(entries, ",")+"]")
	require.NoError(t, LoadApps())

	const readers = 16
	const iters = 500
	done := make(chan struct{}, readers)
	for r := 0; r < readers; r++ {
		go func() {
			for i := 0; i < iters; i++ {
				_, _ = GetAppConfig(fmt.Sprintf("app-%d", i%50))
				_ = ListAppIds()
			}
			done <- struct{}{}
		}()
	}
	for r := 0; r < readers; r++ {
		<-done
	}
	// Test passes if no race was detected (go test -race) and no panic.
}

// -----------------------------------------------------------------------------
// ResetAppsForTest — must actually reset so test isolation holds. If this
// regresses, every other test in the package becomes order-dependent.
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Optional Name field — display label used by the dashboard. Absence must be
// accepted, presence must round-trip, and ListApps must surface it.
// -----------------------------------------------------------------------------

func TestLoadApps_OptionalNameField(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"no-name","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"with-name","name":"Production","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}
    ]`)
	require.NoError(t, LoadApps())

	noName, err := GetAppConfig("no-name")
	require.NoError(t, err)
	assert.Empty(t, noName.Name)

	withName, err := GetAppConfig("with-name")
	require.NoError(t, err)
	assert.Equal(t, "Production", withName.Name)
}

func TestListApps_ReturnsDescriptorsWithName(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"a","name":"App A","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}},
      {"id":"b","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}
    ]`)
	require.NoError(t, LoadApps())

	// Descriptors are returned in unspecified order; reshape into a map so
	// the assertion is stable.
	byId := map[string]AppDescriptor{}
	for _, d := range ListApps() {
		byId[d.Id] = d
	}
	assert.Equal(t, "App A", byId["a"].Name)
	assert.Equal(t, "", byId["b"].Name)
	assert.Len(t, byId, 2)
}

func TestListApps_EmptyWhenNoConfigLoaded(t *testing.T) {
	resetAppsEnv(t)
	assert.Empty(t, ListApps())
}

func TestResetAppsForTest_ClearsRegistry(t *testing.T) {
	resetAppsEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[{"id":"x","accessToken":"t","keys":{"mode":"local","publicPath":"/p","privatePath":"/q"}}]`)
	require.NoError(t, LoadApps())
	assert.NotEmpty(t, ListAppIds())

	ResetAppsForTest()
	assert.Empty(t, ListAppIds())
	_, err := GetAppConfig("x")
	assert.Error(t, err)
}
