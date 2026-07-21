package services

import (
	"context"
	"errors"
	"expo-open-ota/internal/database/postgres/pgdb"
	"expo-open-ota/internal/types"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCliAuthRepo struct {
	keyID           int64
	metadataErr     error
	metadataQueries int
}

func (f *fakeCliAuthRepo) ValidateCliCredential(_ context.Context, _ string, _ types.Auth) (int64, error) {
	return f.keyID, nil
}
func (f *fakeCliAuthRepo) InsertApiKey(_ context.Context, _ string, _ string, _ string, _ string) error {
	return nil
}
func (f *fakeCliAuthRepo) GetApiKeysMetadataByAppID(_ context.Context, _ string) ([]pgdb.GetApiKeysMetadataByAppIDRow, error) {
	f.metadataQueries++
	if f.metadataErr != nil {
		return nil, f.metadataErr
	}
	return []pgdb.GetApiKeysMetadataByAppIDRow{
		{ID: 41, Name: "other-key"},
		{ID: 42, Name: "ci-production"},
	}, nil
}
func (f *fakeCliAuthRepo) RevokeApiKeyByID(_ context.Context, _ int64, _ string) error { return nil }

func TestValidateCliCredentialResolvesKeyIdentity(t *testing.T) {
	repo := &fakeCliAuthRepo{keyID: 42}
	cliAuth := NewCliAuthService(repo, nil)
	cliAuth.SetAuditActive(func() bool { return true })

	credential, err := cliAuth.ValidateCliCredential(context.Background(), "app-1", types.Auth{}, "", netip.Addr{})
	require.NoError(t, err)
	assert.Equal(t, CliCredential{AppID: "app-1", KeyID: "42", KeyName: "ci-production"}, credential)
}

func TestValidateCliCredentialSurvivesMetadataFailure(t *testing.T) {
	// The name only enriches the audit display: an unreadable key list must
	// not fail a valid credential, it just degrades to the app-scope display.
	repo := &fakeCliAuthRepo{keyID: 42, metadataErr: errors.New("list unavailable")}
	cliAuth := NewCliAuthService(repo, nil)
	cliAuth.SetAuditActive(func() bool { return true })

	credential, err := cliAuth.ValidateCliCredential(context.Background(), "app-1", types.Auth{}, "", netip.Addr{})
	require.NoError(t, err)
	assert.Equal(t, CliCredential{AppID: "app-1", KeyID: "42"}, credential)
}

func TestValidateCliCredentialSkipsLookupWhenAuditInactive(t *testing.T) {
	// Community deployments must not pay the metadata query for a name the
	// dropped event would never display.
	repo := &fakeCliAuthRepo{keyID: 42}
	cliAuth := NewCliAuthService(repo, nil)

	credential, err := cliAuth.ValidateCliCredential(context.Background(), "app-1", types.Auth{}, "", netip.Addr{})
	require.NoError(t, err)
	assert.Equal(t, CliCredential{AppID: "app-1", KeyID: "42"}, credential)
	assert.Zero(t, repo.metadataQueries)

	cliAuth.SetAuditActive(func() bool { return false })
	_, err = cliAuth.ValidateCliCredential(context.Background(), "app-1", types.Auth{}, "", netip.Addr{})
	require.NoError(t, err)
	assert.Zero(t, repo.metadataQueries)
}

func TestValidateCliCredentialStatelessHasNoKeyIdentity(t *testing.T) {
	// Stateless mode's credential is the app's Expo token: id 0, no name.
	repo := &fakeCliAuthRepo{keyID: 0}
	cliAuth := NewCliAuthService(repo, nil)
	cliAuth.SetAuditActive(func() bool { return true })

	credential, err := cliAuth.ValidateCliCredential(context.Background(), "app-1", types.Auth{}, "", netip.Addr{})
	require.NoError(t, err)
	assert.Equal(t, CliCredential{AppID: "app-1"}, credential)
	assert.Zero(t, repo.metadataQueries)
}
