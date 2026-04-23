package keyStore

import (
	"expo-open-ota/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two distinct b64-encoded PEM stubs. Contents are not real keys — the
// point of these tests is the appId → key routing, not cryptographic
// behavior. decodeB64 decodes raw bytes, so any non-empty b64 that the
// boot validator accepts will round-trip cleanly here.
const (
	app1PEMB64 = "LS0tLS1CRUdJTiBBUFAgT05FIEtFWS0tLS0tCmtleTEKLS0tLS1FTkQgQVBQIE9ORSBLRVktLS0tLQo="
	app1PEM    = "-----BEGIN APP ONE KEY-----\nkey1\n-----END APP ONE KEY-----\n"

	app2PEMB64 = "LS0tLS1CRUdJTiBBUFAgVFdPIEtFWS0tLS0tCmtleTIKLS0tLS1FTkQgQVBQIFRXTyBLRVktLS0tLQo="
	app2PEM    = "-----BEGIN APP TWO KEY-----\nkey2\n-----END APP TWO KEY-----\n"
)

func resetKeyStoreEnv(t *testing.T) {
	t.Helper()
	vars := []string{"EXPO_APPS_JSON", "EXPO_APP_ID", "EXPO_ACCESS_TOKEN", "KEYS_STORAGE_TYPE", "PUBLIC_EXPO_KEY_B64", "PRIVATE_EXPO_KEY_B64"}
	for _, v := range vars {
		os.Unsetenv(v)
	}
	config.ResetAppsForTest()
	t.Cleanup(func() {
		for _, v := range vars {
			os.Unsetenv(v)
		}
		config.ResetAppsForTest()
	})
}

func TestGetPrivateExpoKey_IsolatedPerApp(t *testing.T) {
	// Multi-app correctness property — two apps served by the same
	// instance must NOT be able to sign with the same private key. A
	// regression that looks up the wrong app would silently cross-
	// contaminate signatures between tenants.
	resetKeyStoreEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"app-1","accessToken":"t1","keys":{"mode":"environment","publicB64":"`+app1PEMB64+`","privateB64":"`+app1PEMB64+`"}},
      {"id":"app-2","accessToken":"t2","keys":{"mode":"environment","publicB64":"`+app2PEMB64+`","privateB64":"`+app2PEMB64+`"}}
    ]`)
	require.NoError(t, config.LoadApps())

	priv1 := GetPrivateExpoKey("app-1")
	priv2 := GetPrivateExpoKey("app-2")

	assert.Equal(t, app1PEM, priv1)
	assert.Equal(t, app2PEM, priv2)
	assert.NotEqual(t, priv1, priv2, "each app must yield its own key")
}

func TestGetPublicExpoKey_IsolatedPerApp(t *testing.T) {
	resetKeyStoreEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"app-1","accessToken":"t1","keys":{"mode":"environment","publicB64":"`+app1PEMB64+`","privateB64":"`+app1PEMB64+`"}},
      {"id":"app-2","accessToken":"t2","keys":{"mode":"environment","publicB64":"`+app2PEMB64+`","privateB64":"`+app2PEMB64+`"}}
    ]`)
	require.NoError(t, config.LoadApps())

	assert.Equal(t, app1PEM, GetPublicExpoKey("app-1"))
	assert.Equal(t, app2PEM, GetPublicExpoKey("app-2"))
}

func TestGetPrivateExpoKey_UnknownAppReturnsEmpty(t *testing.T) {
	// Unknown app id is treated as "no key available" rather than a
	// fatal crash — the handler layer already returns 404 before we get
	// here. Returning "" keeps the key path defensive.
	resetKeyStoreEnv(t)
	os.Setenv("EXPO_APPS_JSON", `[
      {"id":"app-1","accessToken":"t1","keys":{"mode":"environment","publicB64":"`+app1PEMB64+`","privateB64":"`+app1PEMB64+`"}}
    ]`)
	require.NoError(t, config.LoadApps())

	assert.Empty(t, GetPrivateExpoKey("does-not-exist"))
	assert.Empty(t, GetPublicExpoKey("does-not-exist"))
}
