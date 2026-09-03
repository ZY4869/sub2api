package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const backupMultipartPartSize int64 = 8 << 20

func uploadBackupBody(
	ctx context.Context,
	store BackupObjectStore,
	key string,
	body io.Reader,
	contentType string,
) (BackupUploadResult, error) {
	if multipart, ok := store.(BackupMultipartStore); ok {
		result, err := multipart.UploadMultipart(
			ctx,
			key,
			body,
			contentType,
			backupMultipartPartSize,
			nil,
		)
		if err != nil && result.UploadID != "" {
			_ = multipart.AbortMultipart(ctx, key, result.UploadID)
		}
		return result, err
	}

	digest := sha256.New()
	size, err := store.Upload(ctx, key, io.TeeReader(body, digest), contentType)
	if err != nil {
		return BackupUploadResult{}, err
	}
	return BackupUploadResult{
		SizeBytes: size,
		SHA256:    hex.EncodeToString(digest.Sum(nil)),
		PartCount: 1,
	}, nil
}

func verifyBackupChecksum(body io.Reader, expected string) (io.Reader, func() error) {
	expected = normalizeBackupChecksum(expected)
	if expected == "" {
		return body, func() error { return nil }
	}
	digest := sha256.New()
	return io.TeeReader(body, digest), func() error {
		actual := hex.EncodeToString(digest.Sum(nil))
		if actual != expected {
			return fmt.Errorf("%w: expected=%s actual=%s", ErrBackupChecksumMismatch, expected, actual)
		}
		return nil
	}
}

func normalizeBackupChecksum(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return ""
	}
	if _, err := hex.DecodeString(value); err != nil {
		return ""
	}
	return strings.ToLower(value)
}
