package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

// ─── S3 配置管理 ───

func (s *BackupService) GetS3Config(ctx context.Context) (*BackupS3Config, error) {
	cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &BackupS3Config{}, nil
	}
	// 脱敏返回
	cfg.SecretAccessKey = ""
	return cfg, nil
}

func (s *BackupService) UpdateS3Config(ctx context.Context, cfg BackupS3Config) (*BackupS3Config, error) {
	// 如果没提供 secret，保留原有值
	if cfg.SecretAccessKey == "" {
		old, _ := s.loadS3Config(ctx)
		if old != nil {
			cfg.SecretAccessKey = old.SecretAccessKey
		}
	} else {
		// 加密 SecretAccessKey
		encrypted, err := s.encryptor.Encrypt(cfg.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt secret: %w", err)
		}
		cfg.SecretAccessKey = encrypted
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal s3 config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, settingKeyBackupS3Config, string(data)); err != nil {
		return nil, fmt.Errorf("save s3 config: %w", err)
	}

	// 清除缓存的 S3 客户端
	s.storeMu.Lock()
	s.store = nil
	s.s3Cfg = nil
	s.storeMu.Unlock()

	cfg.SecretAccessKey = ""
	return &cfg, nil
}

func (s *BackupService) TestS3Connection(ctx context.Context, cfg BackupS3Config) error {
	if cfg.Bucket == "" || cfg.AccessKeyID == "" {
		return infraerrors.BadRequest("BACKUP_S3_TEST_INCOMPLETE", "bucket and access_key_id are required when testing S3 connectivity")
	}
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return ErrBackupS3TestRequiresSecret
	}
	if err := s.validateS3TestEndpoint(ctx, cfg.Endpoint); err != nil {
		return err
	}

	store, err := s.storeFactory(ctx, &cfg)
	if err != nil {
		return infraerrors.BadRequest("BACKUP_S3_TEST_FAILED", "failed to initialize S3 client for the provided endpoint").WithCause(err)
	}
	if err := store.HeadBucket(ctx); err != nil {
		return infraerrors.BadRequest("BACKUP_S3_TEST_FAILED", "failed to reach the provided S3 bucket with the supplied credentials").WithCause(err)
	}
	return nil
}

func (s *BackupService) loadS3Config(ctx context.Context) (*BackupS3Config, error) {
	raw, err := s.settingRepo.GetValue(ctx, settingKeyBackupS3Config)
	if err != nil || raw == "" {
		return nil, nil //nolint:nilnil // no config is a valid state
	}
	var cfg BackupS3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrBackupS3ConfigCorrupt
	}
	// 解密 SecretAccessKey
	if cfg.SecretAccessKey != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.SecretAccessKey)
		if err != nil {
			// 兼容未加密的旧数据：如果解密失败，保持原值
			logger.LegacyPrintf("service.backup", "[Backup] S3 SecretAccessKey 解密失败（可能是旧的未加密数据）: %v", err)
		} else {
			cfg.SecretAccessKey = decrypted
		}
	}
	return &cfg, nil
}

func (s *BackupService) getOrCreateStore(ctx context.Context, cfg *BackupS3Config) (BackupObjectStore, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if s.store != nil && s.s3Cfg != nil {
		return s.store, nil
	}

	if cfg == nil {
		return nil, ErrBackupS3NotConfigured
	}

	store, err := s.storeFactory(ctx, cfg)
	if err != nil {
		return nil, err
	}
	s.store = store
	s.s3Cfg = cfg
	return store, nil
}

func (s *BackupService) validateS3TestEndpoint(ctx context.Context, endpoint string) error {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return nil
	}

	allowlistCfg := config.URLAllowlistConfig{}
	if s != nil && s.cfg != nil {
		allowlistCfg = s.cfg.Security.URLAllowlist
	}

	allowedHosts := append([]string{}, allowlistCfg.UpstreamHosts...)
	opts := urlvalidator.ValidationOptions{
		AllowedHosts:     allowedHosts,
		AllowPrivate:     false,
		RequireAllowlist: len(allowedHosts) > 0,
	}
	validated, err := urlvalidator.ValidateHTTPSURL(normalized, opts)
	if err != nil {
		return ErrBackupS3EndpointInvalid.WithCause(err)
	}

	host := ""
	if parsed, parseErr := url.Parse(validated); parseErr == nil {
		host = parsed.Hostname()
	}
	if host == "" {
		return ErrBackupS3EndpointInvalid
	}
	if err := urlvalidator.ValidateResolvedIP(host); err != nil {
		return ErrBackupS3EndpointInvalid.WithCause(err)
	}
	_ = ctx
	return nil
}

func (s *BackupService) buildS3Key(cfg *BackupS3Config, fileName string) string {
	prefix := strings.TrimRight(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "backups"
	}
	return fmt.Sprintf("%s/%s/%s", prefix, time.Now().Format("2006/01/02"), fileName)
}

func (s *BackupService) deleteS3Object(ctx context.Context, key string) error {
	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil || s3Cfg == nil {
		return nil
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return err
	}
	return objectStore.Delete(ctx, key)
}
