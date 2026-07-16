package migrations

import (
	"encoding/base64"
	"expo-open-ota/config"
	"expo-open-ota/internal/crypto"
	"expo-open-ota/internal/database/postgres/pgdb"
	"os"
	"path/filepath"
	"testing"
)

// setMasterKey installs a valid 32-byte master key for the duration of the test.
func setMasterKey(t *testing.T) []byte {
	t.Helper()
	raw := []byte("0123456789abcdef0123456789abcdef") // exactly 32 bytes
	t.Setenv("DB_KEYS_MASTER_KEY_B64", base64.StdEncoding.EncodeToString(raw))
	return raw
}

func newKeyPair(t *testing.T) (string, string) {
	t.Helper()
	pub, priv, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	return pub, priv
}

func writeKeyFiles(t *testing.T, pub, priv string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "public-key.pem")
	privPath := filepath.Join(dir, "private-key.pem")
	if err := os.WriteFile(pubPath, []byte(pub), 0o600); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}
	if err := os.WriteFile(privPath, []byte(priv), 0o600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}
	return pubPath, privPath
}

// The whole point of resealing rather than regenerating: expo-updates clients
// pin the certificate at build time, so the migrated app must keep the exact
// same key pair, byte for byte.
func TestSealLegacyKeysIntoDBPreservesLocalKeyPair(t *testing.T) {
	masterKey := setMasterKey(t)
	pub, priv := newKeyPair(t)
	pubPath, privPath := writeKeyFiles(t, pub, priv)

	app := config.AppConfig{
		Id: "11111111-1111-1111-1111-111111111111",
		Keys: config.KeysConfig{
			Mode:        config.KeysModeLocal,
			PublicPath:  pubPath,
			PrivatePath: privPath,
		},
	}
	params := pgdb.MigrateLegacyAppParams{}

	if err := sealLegacyKeysIntoDB(app, &params); err != nil {
		t.Fatalf("expected local keys to seal, got error: %v", err)
	}

	if params.KeysMode == nil || *params.KeysMode != string(config.KeysModeDatabase) {
		t.Fatalf("expected keys_mode=database, got %v", params.KeysMode)
	}
	if params.PathPublicKey != nil || params.PathPrivateKey != nil {
		t.Errorf("expected the stale key paths to be dropped, got %v / %v", params.PathPublicKey, params.PathPrivateKey)
	}
	if params.SealedPublicKey == nil || params.SealedPrivateKey == nil {
		t.Fatal("expected both keys to be sealed")
	}

	unsealedPub, err := crypto.UnsealAESGCM(*params.SealedPublicKey, masterKey)
	if err != nil {
		t.Fatalf("failed to unseal public key: %v", err)
	}
	unsealedPriv, err := crypto.UnsealAESGCM(*params.SealedPrivateKey, masterKey)
	if err != nil {
		t.Fatalf("failed to unseal private key: %v", err)
	}
	if string(unsealedPub) != pub {
		t.Error("public key did not survive the seal/unseal round trip")
	}
	if string(unsealedPriv) != priv {
		t.Error("private key did not survive the seal/unseal round trip")
	}
}

// mode=environment has no column in the apps table, so before this conversion
// it migrated with no key at all and broke signing at the first signature.
func TestSealLegacyKeysIntoDBPreservesEnvironmentKeyPair(t *testing.T) {
	masterKey := setMasterKey(t)
	pub, priv := newKeyPair(t)

	app := config.AppConfig{
		Id: "22222222-2222-2222-2222-222222222222",
		Keys: config.KeysConfig{
			Mode:       config.KeysModeEnvironment,
			PublicB64:  base64.StdEncoding.EncodeToString([]byte(pub)),
			PrivateB64: base64.StdEncoding.EncodeToString([]byte(priv)),
		},
	}
	params := pgdb.MigrateLegacyAppParams{}

	if err := sealLegacyKeysIntoDB(app, &params); err != nil {
		t.Fatalf("expected environment keys to seal, got error: %v", err)
	}
	if params.KeysMode == nil || *params.KeysMode != string(config.KeysModeDatabase) {
		t.Fatalf("expected keys_mode=database, got %v", params.KeysMode)
	}

	unsealedPriv, err := crypto.UnsealAESGCM(*params.SealedPrivateKey, masterKey)
	if err != nil {
		t.Fatalf("failed to unseal private key: %v", err)
	}
	// Sealed content must be the decoded PEM, matching what mode=database
	// stores when the dashboard generates a pair — not the b64 wrapper.
	if string(unsealedPriv) != priv {
		t.Error("private key did not survive the seal/unseal round trip")
	}
}

// The sealed key must still sign, and verify against the sealed public half.
func TestSealedLocalKeyStillSigns(t *testing.T) {
	masterKey := setMasterKey(t)
	pub, priv := newKeyPair(t)
	pubPath, privPath := writeKeyFiles(t, pub, priv)

	app := config.AppConfig{
		Id:   "33333333-3333-3333-3333-333333333333",
		Keys: config.KeysConfig{Mode: config.KeysModeLocal, PublicPath: pubPath, PrivatePath: privPath},
	}
	params := pgdb.MigrateLegacyAppParams{}
	if err := sealLegacyKeysIntoDB(app, &params); err != nil {
		t.Fatalf("failed to seal: %v", err)
	}

	unsealedPriv, err := crypto.UnsealAESGCM(*params.SealedPrivateKey, masterKey)
	if err != nil {
		t.Fatalf("failed to unseal: %v", err)
	}
	if _, err := crypto.SignRSASHA256("some-manifest-body", string(unsealedPriv)); err != nil {
		t.Fatalf("sealed private key can no longer sign: %v", err)
	}
}

// aws-secrets-manager is a reference that stays reachable from every replica,
// so it must pass through the migration untouched.
func TestSealLegacyKeysIntoDBLeavesAWSSecretsManagerAlone(t *testing.T) {
	setMasterKey(t)
	secretPub, secretPriv := "expo-public-secret-id", "expo-private-secret-id"
	app := config.AppConfig{
		Id: "44444444-4444-4444-4444-444444444444",
		Keys: config.KeysConfig{
			Mode:            config.KeysModeAWSSM,
			PublicSecretId:  secretPub,
			PrivateSecretId: secretPriv,
		},
	}
	params := pgdb.MigrateLegacyAppParams{
		AwsSecretIDPublic:  &secretPub,
		AwsSecretIDPrivate: &secretPriv,
	}

	if err := sealLegacyKeysIntoDB(app, &params); err != nil {
		t.Fatalf("expected aws-sm to be a no-op, got error: %v", err)
	}
	if params.SealedPublicKey != nil || params.SealedPrivateKey != nil {
		t.Error("aws-sm app must not have keys sealed into the database")
	}
	if params.AwsSecretIDPublic == nil || *params.AwsSecretIDPublic != secretPub {
		t.Error("aws-sm secret ids must survive the migration untouched")
	}
}

// Failing loudly beats migrating an empty key and breaking signing at runtime.
func TestSealLegacyKeysIntoDBFailsWhenKeysAreUnreadable(t *testing.T) {
	setMasterKey(t)
	app := config.AppConfig{
		Id: "55555555-5555-5555-5555-555555555555",
		Keys: config.KeysConfig{
			Mode:        config.KeysModeLocal,
			PublicPath:  filepath.Join(t.TempDir(), "does-not-exist.pem"),
			PrivatePath: filepath.Join(t.TempDir(), "missing-too.pem"),
		},
	}
	params := pgdb.MigrateLegacyAppParams{}

	if err := sealLegacyKeysIntoDB(app, &params); err == nil {
		t.Fatal("expected the migration to abort on unreadable key files, got nil")
	}
	if params.SealedPrivateKey != nil {
		t.Error("no key should have been sealed on the failure path")
	}
}

func TestSealLegacyKeysIntoDBFailsWithoutMasterKey(t *testing.T) {
	t.Setenv("DB_KEYS_MASTER_KEY_B64", "")
	pub, priv := newKeyPair(t)
	pubPath, privPath := writeKeyFiles(t, pub, priv)

	app := config.AppConfig{
		Id:   "66666666-6666-6666-6666-666666666666",
		Keys: config.KeysConfig{Mode: config.KeysModeLocal, PublicPath: pubPath, PrivatePath: privPath},
	}
	params := pgdb.MigrateLegacyAppParams{}

	if err := sealLegacyKeysIntoDB(app, &params); err == nil {
		t.Fatal("expected the migration to abort without a master key, got nil")
	}
}
