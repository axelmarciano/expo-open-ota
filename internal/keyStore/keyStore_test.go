package keyStore

import (
	"expo-open-ota/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Two distinct b64-encoded PEM stubs. Contents are not real keys — the point
// of these tests is that each app's own KeysConfig drives key resolution, not
// cryptographic behavior. decodeB64 decodes raw bytes, so any non-empty b64
// round-trips cleanly here.
const (
	app1PEMB64 = "LS0tLS1CRUdJTiBBUFAgT05FIEtFWS0tLS0tCmtleTEKLS0tLS1FTkQgQVBQIE9ORSBLRVktLS0tLQo="
	app1PEM    = "-----BEGIN APP ONE KEY-----\nkey1\n-----END APP ONE KEY-----\n"

	app2PEMB64 = "LS0tLS1CRUdJTiBBUFAgVFdPIEtFWS0tLS0tCmtleTIKLS0tLS1FTkQgQVBQIFRXTyBLRVktLS0tLQo="
	app2PEM    = "-----BEGIN APP TWO KEY-----\nkey2\n-----END APP TWO KEY-----\n"
)

func envKeysApp(id, pubB64, privB64 string) config.AppConfig {
	return config.AppConfig{
		Id:          id,
		AccessToken: "t",
		Keys: config.KeysConfig{
			Mode:       config.KeysModeEnvironment,
			PublicB64:  pubB64,
			PrivateB64: privB64,
		},
	}
}

func TestGetPrivateExpoKey_IsolatedPerApp(t *testing.T) {
	// Multi-app correctness property — two apps served by the same instance
	// must NOT sign with the same private key. A regression that resolved the
	// wrong app's KeysConfig would silently cross-contaminate signatures
	// between tenants.
	app1 := envKeysApp("app-1", app1PEMB64, app1PEMB64)
	app2 := envKeysApp("app-2", app2PEMB64, app2PEMB64)

	priv1 := GetPrivateExpoKey(app1)
	priv2 := GetPrivateExpoKey(app2)

	assert.Equal(t, app1PEM, priv1)
	assert.Equal(t, app2PEM, priv2)
	assert.NotEqual(t, priv1, priv2, "each app must yield its own key")
}

func TestGetPublicExpoKey_IsolatedPerApp(t *testing.T) {
	app1 := envKeysApp("app-1", app1PEMB64, app1PEMB64)
	app2 := envKeysApp("app-2", app2PEMB64, app2PEMB64)

	assert.Equal(t, app1PEM, GetPublicExpoKey(app1))
	assert.Equal(t, app2PEM, GetPublicExpoKey(app2))
}

func TestGetExpoKey_UnconfiguredKeysReturnEmpty(t *testing.T) {
	// An app whose KeysConfig carries no material is treated as "no key
	// available" rather than a crash — the handler layer already returns 404
	// for unknown apps before we get here, so returning "" keeps the key path
	// defensive.
	app := config.AppConfig{Id: "app-1", AccessToken: "t", Keys: config.KeysConfig{Mode: config.KeysModeEnvironment}}
	assert.Empty(t, GetPrivateExpoKey(app))
	assert.Empty(t, GetPublicExpoKey(app))
}
