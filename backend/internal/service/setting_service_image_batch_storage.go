package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	ImageBatchStorageBackendDB    = "db"
	ImageBatchStorageBackendLocal = "local"
	ImageBatchStorageBackendS3    = "s3"
	ImageBatchStorageBackendR2    = "r2"
)

var (
	ErrImageBatchStorageSettingsInvalid = infraerrors.BadRequest("IMAGE_BATCH_STORAGE_SETTINGS_INVALID", "image batch storage settings are invalid")
	ErrImageBatchStorageSecretInvalid   = infraerrors.BadRequest("IMAGE_BATCH_STORAGE_SECRET_INVALID", "image batch storage secret cannot be decrypted safely")
	ErrImageBatchStorageTestSecret      = infraerrors.BadRequest("IMAGE_BATCH_STORAGE_TEST_SECRET_REQUIRED", "secret_access_key is required when testing image batch storage")
	ErrImageBatchStorageEndpointInvalid = infraerrors.BadRequest("IMAGE_BATCH_STORAGE_ENDPOINT_INVALID", "image batch storage endpoint is not allowed")
)

type ImageBatchStorageSettings struct {
	Backend                   string `json:"backend"`
	Endpoint                  string `json:"endpoint"`
	Bucket                    string `json:"bucket"`
	Prefix                    string `json:"prefix"`
	Region                    string `json:"region"`
	ForcePathStyle            bool   `json:"force_path_style"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"secret_access_key,omitempty"`
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"`
}

func DefaultImageBatchStorageSettingsFromConfig(cfg *config.Config) *ImageBatchStorageSettings {
	settings := &ImageBatchStorageSettings{
		Backend: ImageBatchStorageBackendLocal,
		Prefix:  "image-batches",
		Region:  "auto",
	}
	if cfg == nil {
		return settings
	}
	storage := cfg.ImageBatch.Storage
	settings.Backend = storage.Backend
	settings.Endpoint = storage.Endpoint
	settings.Bucket = storage.Bucket
	settings.Prefix = storage.Prefix
	settings.Region = storage.Region
	settings.ForcePathStyle = storage.ForcePathStyle
	settings.AccessKeyID = storage.AccessKeyID
	settings.SecretAccessKey = storage.SecretAccessKey
	settings.SecretAccessKeyConfigured = strings.TrimSpace(storage.SecretAccessKey) != ""
	return NormalizeImageBatchStorageSettings(settings)
}

func NormalizeImageBatchStorageSettings(input *ImageBatchStorageSettings) *ImageBatchStorageSettings {
	defaults := &ImageBatchStorageSettings{
		Backend: ImageBatchStorageBackendLocal,
		Prefix:  "image-batches",
		Region:  "auto",
	}
	if input == nil {
		return defaults
	}
	out := *input
	out.Backend = normalizeImageBatchStorageBackend(out.Backend)
	out.Endpoint = strings.TrimSpace(out.Endpoint)
	out.Bucket = strings.TrimSpace(out.Bucket)
	out.Prefix = strings.Trim(strings.TrimSpace(out.Prefix), "/")
	if out.Prefix == "" {
		out.Prefix = defaults.Prefix
	}
	out.Region = strings.TrimSpace(out.Region)
	if out.Region == "" {
		out.Region = defaults.Region
	}
	out.AccessKeyID = strings.TrimSpace(out.AccessKeyID)
	out.SecretAccessKey = strings.TrimSpace(out.SecretAccessKey)
	out.SecretAccessKeyConfigured = out.SecretAccessKeyConfigured || out.SecretAccessKey != ""
	return &out
}

func normalizeImageBatchStorageBackend(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ImageBatchStorageBackendS3:
		return ImageBatchStorageBackendS3
	case ImageBatchStorageBackendR2:
		return ImageBatchStorageBackendR2
	case ImageBatchStorageBackendDB, ImageBatchStorageBackendLocal, "":
		return ImageBatchStorageBackendLocal
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func ValidateImageBatchStorageSettings(settings *ImageBatchStorageSettings, requireSecret bool) error {
	normalized := NormalizeImageBatchStorageSettings(settings)
	switch normalized.Backend {
	case ImageBatchStorageBackendLocal, ImageBatchStorageBackendDB:
		return nil
	case ImageBatchStorageBackendS3, ImageBatchStorageBackendR2:
		if normalized.Bucket == "" {
			return ErrImageBatchStorageSettingsInvalid.WithCause(fmt.Errorf("bucket is required"))
		}
		if normalized.AccessKeyID == "" {
			return ErrImageBatchStorageSettingsInvalid.WithCause(fmt.Errorf("access_key_id is required"))
		}
		if requireSecret && normalized.SecretAccessKey == "" {
			return ErrImageBatchStorageTestSecret
		}
		if normalized.Endpoint != "" {
			if _, err := url.Parse(normalized.Endpoint); err != nil {
				return ErrImageBatchStorageEndpointInvalid.WithCause(err)
			}
		}
	default:
		return ErrImageBatchStorageSettingsInvalid.WithCause(fmt.Errorf("unsupported backend: %s", normalized.Backend))
	}
	return nil
}

func (s *SettingService) GetImageBatchStorageSettings(ctx context.Context) (*ImageBatchStorageSettings, error) {
	settings, err := s.getImageBatchStorageSettings(ctx, false)
	if err != nil {
		return nil, err
	}
	settings.SecretAccessKey = ""
	return settings, nil
}

func (s *SettingService) getImageBatchStorageSettings(ctx context.Context, includeSecret bool) (*ImageBatchStorageSettings, error) {
	defaults := DefaultImageBatchStorageSettingsFromConfig(s.configOrNil())
	if s == nil || s.settingRepo == nil {
		if !includeSecret {
			defaults.SecretAccessKey = ""
		}
		return defaults, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyImageBatchStorageSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			if includeSecret && defaults.SecretAccessKey != "" {
				return defaults, nil
			}
			if !includeSecret {
				defaults.SecretAccessKey = ""
			}
			return defaults, nil
		}
		return nil, fmt.Errorf("get image batch storage settings: %w", err)
	}
	var settings ImageBatchStorageSettings
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return nil, ErrImageBatchStorageSettingsInvalid.WithCause(err)
		}
	}
	normalized := NormalizeImageBatchStorageSettings(&settings)
	if normalized.SecretAccessKey != "" {
		secret, err := s.decryptImageBatchStorageSecret(normalized.SecretAccessKey)
		if err != nil {
			return nil, ErrImageBatchStorageSecretInvalid.WithCause(err)
		}
		normalized.SecretAccessKey = secret
		normalized.SecretAccessKeyConfigured = true
	}
	if !includeSecret {
		normalized.SecretAccessKey = ""
	}
	return normalized, nil
}

func (s *SettingService) SetImageBatchStorageSettings(ctx context.Context, settings *ImageBatchStorageSettings) (*ImageBatchStorageSettings, error) {
	normalized := NormalizeImageBatchStorageSettings(settings)
	oldEncrypted := ""
	validationSecret := normalized.SecretAccessKey
	if normalized.SecretAccessKey == "" {
		old, err := s.getImageBatchStorageSettingsEncrypted(ctx)
		if err != nil {
			return nil, err
		}
		oldEncrypted = strings.TrimSpace(old.SecretAccessKey)
		normalized.SecretAccessKeyConfigured = oldEncrypted != ""
		if oldEncrypted != "" {
			secret, err := s.decryptImageBatchStorageSecret(oldEncrypted)
			if err != nil {
				return nil, ErrImageBatchStorageSecretInvalid.WithCause(err)
			}
			validationSecret = secret
		}
	} else {
		encrypted, err := s.encryptImageBatchStorageSecret(normalized.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt image batch storage secret: %w", err)
		}
		oldEncrypted = encrypted
		normalized.SecretAccessKeyConfigured = true
	}
	validation := *normalized
	validation.SecretAccessKey = validationSecret
	if err := ValidateImageBatchStorageSettings(&validation, validation.Backend == ImageBatchStorageBackendS3 || validation.Backend == ImageBatchStorageBackendR2); err != nil {
		return nil, err
	}
	stored := *normalized
	stored.SecretAccessKey = oldEncrypted
	data, err := json.Marshal(&stored)
	if err != nil {
		return nil, fmt.Errorf("marshal image batch storage settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyImageBatchStorageSettings, string(data)); err != nil {
		return nil, fmt.Errorf("save image batch storage settings: %w", err)
	}
	s.notifyUpdateCallbacks()
	normalized.SecretAccessKey = ""
	return normalized, nil
}

func (s *SettingService) TestImageBatchStorageSettings(ctx context.Context, settings *ImageBatchStorageSettings) error {
	normalized := NormalizeImageBatchStorageSettings(settings)
	if normalized.SecretAccessKey == "" {
		current, err := s.getImageBatchStorageSettings(ctx, true)
		if err == nil && current != nil {
			normalized.SecretAccessKey = current.SecretAccessKey
		}
	}
	if err := ValidateImageBatchStorageSettings(normalized, normalized.Backend == ImageBatchStorageBackendS3 || normalized.Backend == ImageBatchStorageBackendR2); err != nil {
		return err
	}
	if normalized.Backend == ImageBatchStorageBackendLocal || normalized.Backend == ImageBatchStorageBackendDB {
		return nil
	}
	if err := s.validateImageBatchStorageEndpoint(ctx, normalized.Endpoint); err != nil {
		return err
	}
	store, err := newS3LikeObjectStore(ctx, normalized)
	if err != nil {
		return ErrImageBatchStorageSettingsInvalid.WithCause(err)
	}
	if err := store.HeadBucket(ctx); err != nil {
		return ErrImageBatchStorageSettingsInvalid.WithCause(err)
	}
	return nil
}

func (s *SettingService) getImageBatchStorageSettingsEncrypted(ctx context.Context) (*ImageBatchStorageSettings, error) {
	defaults := DefaultImageBatchStorageSettingsFromConfig(s.configOrNil())
	if defaults.SecretAccessKey != "" {
		encrypted, err := s.encryptImageBatchStorageSecret(defaults.SecretAccessKey)
		if err != nil {
			return nil, err
		}
		defaults.SecretAccessKey = encrypted
		return defaults, nil
	}
	if s == nil || s.settingRepo == nil {
		return defaults, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyImageBatchStorageSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaults, nil
		}
		return nil, err
	}
	var settings ImageBatchStorageSettings
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &settings); err != nil {
			return nil, ErrImageBatchStorageSettingsInvalid.WithCause(err)
		}
	}
	if strings.TrimSpace(settings.SecretAccessKey) != "" {
		if _, err := s.decryptImageBatchStorageSecret(settings.SecretAccessKey); err != nil {
			return nil, ErrImageBatchStorageSecretInvalid.WithCause(err)
		}
	}
	return NormalizeImageBatchStorageSettings(&settings), nil
}

func (s *SettingService) encryptImageBatchStorageSecret(secret string) (string, error) {
	key, err := s.imageBatchStorageSecretKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(secret), nil)), nil
}

func (s *SettingService) decryptImageBatchStorageSecret(ciphertext string) (string, error) {
	key, err := s.imageBatchStorageSecretKey()
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *SettingService) imageBatchStorageSecretKey() ([]byte, error) {
	if s == nil || s.cfg == nil || strings.TrimSpace(s.cfg.Totp.EncryptionKey) == "" {
		return nil, fmt.Errorf("totp encryption key is not configured")
	}
	key, err := hex.DecodeString(strings.TrimSpace(s.cfg.Totp.EncryptionKey))
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("totp encryption key must be 32 bytes")
	}
	return key, nil
}

func (s *SettingService) validateImageBatchStorageEndpoint(ctx context.Context, endpoint string) error {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return nil
	}
	allowlistCfg := config.URLAllowlistConfig{}
	if s != nil && s.cfg != nil {
		allowlistCfg = s.cfg.Security.URLAllowlist
	}
	opts := urlvalidator.ValidationOptions{
		AllowedHosts:     append([]string{}, allowlistCfg.UpstreamHosts...),
		AllowPrivate:     false,
		RequireAllowlist: len(allowlistCfg.UpstreamHosts) > 0,
	}
	validated, err := urlvalidator.ValidateHTTPSURL(normalized, opts)
	if err != nil {
		return ErrImageBatchStorageEndpointInvalid.WithCause(err)
	}
	host := ""
	if parsed, parseErr := url.Parse(validated); parseErr == nil {
		host = parsed.Hostname()
	}
	if host == "" {
		return ErrImageBatchStorageEndpointInvalid
	}
	if err := urlvalidator.ValidateResolvedIP(host); err != nil {
		return ErrImageBatchStorageEndpointInvalid.WithCause(err)
	}
	_ = ctx
	return nil
}

type imageBatchObjectStore interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (int64, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	HeadBucket(ctx context.Context) error
}

type s3LikeObjectStore struct {
	client *s3.Client
	bucket string
}

func newS3LikeObjectStore(ctx context.Context, cfg *ImageBatchStorageSettings) (imageBatchObjectStore, error) {
	if cfg == nil {
		return nil, ErrImageBatchStorageSettingsInvalid
	}
	region := cfg.Region
	if region == "" {
		region = "auto"
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = &cfg.Endpoint
		}
		if cfg.ForcePathStyle {
			o.UsePathStyle = true
		}
		o.APIOptions = append(o.APIOptions, v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	return &s3LikeObjectStore{client: client, bucket: cfg.Bucket}, nil
}

func (s *s3LikeObjectStore) Upload(ctx context.Context, key string, body io.Reader, contentType string) (int64, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	return int64(len(data)), err
}

func (s *s3LikeObjectStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (s *s3LikeObjectStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

func (s *s3LikeObjectStore) HeadBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	return err
}

func imageBatchStorageObjectKey(settings *ImageBatchStorageSettings, jobID string, customID string, sha string, contentType string) string {
	prefix := "image-batches"
	if settings != nil && strings.TrimSpace(settings.Prefix) != "" {
		prefix = strings.Trim(strings.TrimSpace(settings.Prefix), "/")
	}
	return fmt.Sprintf("%s/%s/%s/%s%s", prefix, time.Now().UTC().Format("2006/01/02"), safeObjectPathSegment(jobID), safeObjectPathSegment(customID+"-"+sha), imageBatchContentTypeExt(contentType))
}

func safeObjectPathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "output"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '#', '%':
			return '-'
		default:
			return r
		}
	}, value)
}

func imageBatchContentTypeExt(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}
