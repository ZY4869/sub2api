//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestImageBatchStorageSettingsEncryptsAndRedactsSecret(t *testing.T) {
	repo := &settingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, imageBatchStorageTestConfig())

	got, err := svc.SetImageBatchStorageSettings(context.Background(), &ImageBatchStorageSettings{
		Backend:         ImageBatchStorageBackendS3,
		Endpoint:        "https://s3.example.com",
		Bucket:          "images",
		Prefix:          "/async/",
		Region:          "",
		ForcePathStyle:  true,
		AccessKeyID:     "ak",
		SecretAccessKey: "secret",
	})
	require.NoError(t, err)
	require.Equal(t, ImageBatchStorageBackendS3, got.Backend)
	require.Equal(t, "async", got.Prefix)
	require.True(t, got.SecretAccessKeyConfigured)
	require.Empty(t, got.SecretAccessKey)

	raw := repo.values[SettingKeyImageBatchStorageSettings]
	require.NotContains(t, raw, `"secret_access_key":"secret"`)
	var stored ImageBatchStorageSettings
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	require.NotEmpty(t, stored.SecretAccessKey)

	loaded, err := svc.getImageBatchStorageSettings(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, "secret", loaded.SecretAccessKey)
}

func TestImageBatchStorageSettingsPreservesExistingSecret(t *testing.T) {
	repo := &settingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, imageBatchStorageTestConfig())
	_, err := svc.SetImageBatchStorageSettings(context.Background(), &ImageBatchStorageSettings{
		Backend:         ImageBatchStorageBackendS3,
		Endpoint:        "https://s3.example.com",
		Bucket:          "images",
		AccessKeyID:     "ak",
		SecretAccessKey: "secret",
	})
	require.NoError(t, err)
	firstRaw := repo.values[SettingKeyImageBatchStorageSettings]

	updated, err := svc.SetImageBatchStorageSettings(context.Background(), &ImageBatchStorageSettings{
		Backend:     ImageBatchStorageBackendS3,
		Endpoint:    "https://s3.example.com",
		Bucket:      "images-2",
		AccessKeyID: "ak",
	})
	require.NoError(t, err)
	require.True(t, updated.SecretAccessKeyConfigured)
	require.Empty(t, updated.SecretAccessKey)

	var before, after ImageBatchStorageSettings
	require.NoError(t, json.Unmarshal([]byte(firstRaw), &before))
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyImageBatchStorageSettings]), &after))
	require.Equal(t, before.SecretAccessKey, after.SecretAccessKey)
}

func TestImageBatchStorageSettingsRejectsUndecryptableStoredSecret(t *testing.T) {
	repo := &settingRepoStub{values: map[string]string{}}
	raw, err := json.Marshal(ImageBatchStorageSettings{
		Backend:         ImageBatchStorageBackendS3,
		Endpoint:        "https://s3.example.com",
		Bucket:          "images",
		AccessKeyID:     "ak",
		SecretAccessKey: "not-base64",
	})
	require.NoError(t, err)
	repo.values[SettingKeyImageBatchStorageSettings] = string(raw)
	svc := NewSettingService(repo, imageBatchStorageTestConfig())

	_, err = svc.SetImageBatchStorageSettings(context.Background(), &ImageBatchStorageSettings{
		Backend:     ImageBatchStorageBackendS3,
		Endpoint:    "https://s3.example.com",
		Bucket:      "images",
		AccessKeyID: "ak",
	})
	require.ErrorIs(t, err, ErrImageBatchStorageSecretInvalid)
	require.True(t, strings.Contains(repo.values[SettingKeyImageBatchStorageSettings], "not-base64"))
}

func imageBatchStorageTestConfig() *config.Config {
	return &config.Config{
		Totp: config.TotpConfig{
			EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
	}
}
