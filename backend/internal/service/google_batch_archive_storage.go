package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	googleBatchArchiveResultFilename   = "result.jsonl"
	googleBatchArchiveSnapshotFilename = "batch_snapshot.json"
	googleBatchArchiveManifestFilename = "manifest.json"
)

type GoogleBatchArchiveStorage struct{}

func NewGoogleBatchArchiveStorage() *GoogleBatchArchiveStorage {
	return &GoogleBatchArchiveStorage{}
}

func normalizeFileStorageRoot(root string) string {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return googleBatchArchiveDefaultLocalStorageRoot
	}
	cleaned := filepath.Clean(trimmed)
	if filepath.IsAbs(cleaned) {
		return cleaned
	}
	if absRoot, err := filepath.Abs(cleaned); err == nil {
		return absRoot
	}
	return cleaned
}

func (s *GoogleBatchArchiveStorage) StoreBytes(_ context.Context, settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob, filename string, payload []byte) (string, int64, string, error) {
	if job == nil {
		return "", 0, "", fmt.Errorf("archive job is nil")
	}
	dir, err := s.ensureJobDir(settings, job)
	if err != nil {
		return "", 0, "", err
	}
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", 0, "", fmt.Errorf("archive filename is required")
	}
	localPath := filepath.Join(dir, name)
	if err := os.WriteFile(localPath, payload, 0o644); err != nil {
		return "", 0, "", err
	}
	sum := sha256.Sum256(payload)
	relativePath, err := s.relativePath(settings, localPath)
	if err != nil {
		return "", 0, "", err
	}
	return relativePath, int64(len(payload)), hex.EncodeToString(sum[:]), nil
}

func (s *GoogleBatchArchiveStorage) OpenReader(settings *GoogleBatchArchiveSettings, relativePath string) (*os.File, os.FileInfo, error) {
	localPath, err := s.resolveLocalPath(settings, relativePath)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(localPath)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return file, info, nil
}

func (s *GoogleBatchArchiveStorage) ReadAll(settings *GoogleBatchArchiveSettings, relativePath string) ([]byte, error) {
	file, _, err := s.OpenReader(settings, relativePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(file)
}

func (s *GoogleBatchArchiveStorage) DeleteRelativePath(settings *GoogleBatchArchiveSettings, relativePath string) error {
	localPath, err := s.resolveLocalPath(settings, relativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *GoogleBatchArchiveStorage) DeleteJobDir(settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob) error {
	if job == nil {
		return nil
	}
	dir, err := s.jobDir(settings, job)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return nil
}

func (s *GoogleBatchArchiveStorage) ensureJobDir(settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob) (string, error) {
	dir, err := s.jobDir(settings, job)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *GoogleBatchArchiveStorage) jobDir(settings *GoogleBatchArchiveSettings, job *GoogleBatchArchiveJob) (string, error) {
	if job == nil {
		return "", fmt.Errorf("archive job is nil")
	}
	root := normalizeFileStorageRoot(settingsRoot(settings))
	createdAt := job.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if job.ID <= 0 {
		return "", fmt.Errorf("archive job id is required")
	}
	return filepath.Join(
		root,
		createdAt.UTC().Format("2006"),
		createdAt.UTC().Format("01"),
		createdAt.UTC().Format("02"),
		fmt.Sprintf("%d", job.ID),
	), nil
}

func (s *GoogleBatchArchiveStorage) relativePath(settings *GoogleBatchArchiveSettings, absolutePath string) (string, error) {
	root := normalizeFileStorageRoot(settingsRoot(settings))
	rel, err := filepath.Rel(root, absolutePath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func (s *GoogleBatchArchiveStorage) resolveLocalPath(settings *GoogleBatchArchiveSettings, relativePath string) (string, error) {
	root := normalizeFileStorageRoot(settingsRoot(settings))
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return "", fmt.Errorf("relative_path is required")
	}
	cleaned := filepath.Clean(filepath.FromSlash(trimmed))
	localPath := filepath.Join(root, cleaned)
	if rel, err := filepath.Rel(root, localPath); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid archive relative path")
	}
	return localPath, nil
}

func settingsRoot(settings *GoogleBatchArchiveSettings) string {
	if settings == nil {
		return googleBatchArchiveDefaultLocalStorageRoot
	}
	return settings.LocalStorageRoot
}
