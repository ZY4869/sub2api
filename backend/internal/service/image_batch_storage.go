package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const imageBatchOutputMetadataStorageKey = "storage_key"

func (s *ImageBatchService) prepareOutputForStorage(ctx context.Context, output ImageBatchOutput) (ImageBatchOutput, error) {
	settings, err := s.imageBatchStorageSettings(ctx)
	if err != nil {
		return output, err
	}
	if !imageBatchStorageExternal(settings) {
		output.StorageBackend = ImageBatchStorageBackendDB
		return output, nil
	}
	if len(output.Content) == 0 {
		return output, ErrImageBatchOutputNotFound
	}
	store, err := s.imageBatchObjectStore(ctx, settings)
	if err != nil {
		return output, err
	}
	key := imageBatchStorageObjectKey(settings, output.JobID, output.CustomID, output.SHA256, output.ContentType)
	size, err := store.Upload(ctx, key, bytes.NewReader(output.Content), firstNonEmptyString(output.ContentType, "image/png"))
	if err != nil {
		return output, fmt.Errorf("upload image batch output: %w", err)
	}
	if output.Metadata == nil {
		output.Metadata = map[string]any{}
	}
	output.Metadata[imageBatchOutputMetadataStorageKey] = key
	output.StorageBackend = settings.Backend
	output.Content = nil
	if size > 0 {
		output.SizeBytes = size
	}
	return output, nil
}

func (s *ImageBatchService) loadOutputContent(ctx context.Context, output ImageBatchOutput) ([]byte, error) {
	switch normalizeImageBatchStorageBackend(output.StorageBackend) {
	case ImageBatchStorageBackendS3, ImageBatchStorageBackendR2:
		key := imageBatchOutputStorageKey(output)
		if key == "" {
			return nil, ErrImageBatchOutputNotFound
		}
		settings, err := s.imageBatchStorageSettings(ctx)
		if err != nil {
			return nil, err
		}
		if !imageBatchStorageExternal(settings) {
			return nil, ErrImageBatchOutputNotFound
		}
		store, err := s.imageBatchObjectStore(ctx, settings)
		if err != nil {
			return nil, err
		}
		reader, err := store.Download(ctx, key)
		if err != nil {
			return nil, err
		}
		defer func() { _ = reader.Close() }()
		return io.ReadAll(reader)
	default:
		if len(output.Content) == 0 {
			return nil, ErrImageBatchOutputNotFound
		}
		return append([]byte(nil), output.Content...), nil
	}
}

func (s *ImageBatchService) deleteOutputObjectsBestEffort(ctx context.Context, outputs []ImageBatchOutput) {
	if len(outputs) == 0 {
		return
	}
	settings, err := s.imageBatchStorageSettings(ctx)
	if err != nil || !imageBatchStorageExternal(settings) {
		return
	}
	store, err := s.imageBatchObjectStore(ctx, settings)
	if err != nil {
		return
	}
	for _, output := range outputs {
		key := imageBatchOutputStorageKey(output)
		if key == "" {
			continue
		}
		_ = store.Delete(ctx, key)
	}
}

func (s *ImageBatchService) imageBatchStorageSettings(ctx context.Context) (*ImageBatchStorageSettings, error) {
	if s == nil || s.settingService == nil {
		return DefaultImageBatchStorageSettingsFromConfig(s.configOrNil()), nil
	}
	return s.settingService.getImageBatchStorageSettings(ctx, true)
}

func (s *ImageBatchService) imageBatchObjectStore(ctx context.Context, settings *ImageBatchStorageSettings) (imageBatchObjectStore, error) {
	if settings == nil {
		return nil, ErrImageBatchStorageSettingsInvalid
	}
	s.storageMu.Lock()
	defer s.storageMu.Unlock()
	if s.storageStore != nil && imageBatchStorageSettingsEquivalent(s.storageCfg, settings) {
		return s.storageStore, nil
	}
	store, err := newS3LikeObjectStore(ctx, settings)
	if err != nil {
		return nil, err
	}
	s.storageStore = store
	copyCfg := *settings
	s.storageCfg = &copyCfg
	return store, nil
}

func (s *ImageBatchService) clearImageBatchObjectStoreCache() {
	if s == nil {
		return
	}
	s.storageMu.Lock()
	defer s.storageMu.Unlock()
	s.storageStore = nil
	s.storageCfg = nil
}

func (s *ImageBatchService) configOrNil() *config.Config {
	if s == nil {
		return nil
	}
	return s.cfg
}

func imageBatchStorageExternal(settings *ImageBatchStorageSettings) bool {
	if settings == nil {
		return false
	}
	backend := normalizeImageBatchStorageBackend(settings.Backend)
	return backend == ImageBatchStorageBackendS3 || backend == ImageBatchStorageBackendR2
}

func imageBatchOutputStorageKey(output ImageBatchOutput) string {
	if output.Metadata == nil {
		return ""
	}
	value, _ := output.Metadata[imageBatchOutputMetadataStorageKey].(string)
	return strings.TrimSpace(value)
}

func imageBatchStorageSettingsEquivalent(a *ImageBatchStorageSettings, b *ImageBatchStorageSettings) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Backend == b.Backend &&
		a.Endpoint == b.Endpoint &&
		a.Bucket == b.Bucket &&
		a.Prefix == b.Prefix &&
		a.Region == b.Region &&
		a.ForcePathStyle == b.ForcePathStyle &&
		a.AccessKeyID == b.AccessKeyID &&
		a.SecretAccessKey == b.SecretAccessKey
}
