//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeImageBatchObjectStore struct {
	objects map[string][]byte
}

func (s *fakeImageBatchObjectStore) Upload(_ context.Context, key string, body io.Reader, _ string) (int64, error) {
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	s.objects[key] = append([]byte(nil), data...)
	return int64(len(data)), nil
}

func (s *fakeImageBatchObjectStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.objects[key])), nil
}

func (s *fakeImageBatchObjectStore) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *fakeImageBatchObjectStore) HeadBucket(context.Context) error {
	return nil
}

func TestImageBatchStorageExternalBackendMovesContentToObjectStore(t *testing.T) {
	store := &fakeImageBatchObjectStore{objects: map[string][]byte{}}
	svc := &ImageBatchService{
		storageStore: store,
		storageCfg: &ImageBatchStorageSettings{
			Backend:         ImageBatchStorageBackendS3,
			Endpoint:        "https://s3.example.com",
			Bucket:          "images",
			Prefix:          "async",
			Region:          "auto",
			AccessKeyID:     "ak",
			SecretAccessKey: "secret",
		},
		settingService: NewSettingService(&settingRepoStub{values: map[string]string{}}, imageBatchStorageTestConfig()),
	}
	_, err := svc.settingService.SetImageBatchStorageSettings(context.Background(), svc.storageCfg)
	require.NoError(t, err)

	stored, err := svc.prepareOutputForStorage(context.Background(), ImageBatchOutput{
		JobID:          "job-1",
		CustomID:       "item-1",
		ContentType:    "image/png",
		StorageBackend: ImageBatchStorageBackendDB,
		Content:        []byte("image-bytes"),
		SizeBytes:      int64(len("image-bytes")),
		SHA256:         "sha",
	})
	require.NoError(t, err)
	require.Equal(t, ImageBatchStorageBackendS3, stored.StorageBackend)
	require.Empty(t, stored.Content)
	key := imageBatchOutputStorageKey(stored)
	require.NotEmpty(t, key)
	require.Equal(t, []byte("image-bytes"), store.objects[key])

	body, err := svc.loadOutputContent(context.Background(), stored)
	require.NoError(t, err)
	require.Equal(t, []byte("image-bytes"), body)
}
