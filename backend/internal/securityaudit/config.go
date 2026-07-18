package securityaudit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

type configState struct {
	storage storageConfig
	active  ActiveConfig
	public  PublicConfig
	err     string
	loaded  time.Time
}

type ConfigManager struct {
	settings    service.SettingRepository
	encryptor   service.SecretEncryptor
	invalidator ConfigInvalidator
	state       atomic.Value
}

func NewConfigManager(settings service.SettingRepository, encryptor service.SecretEncryptor) *ConfigManager {
	m := &ConfigManager{settings: settings, encryptor: encryptor}
	m.state.Store(configState{storage: defaultStorageConfig(), public: publicFromStorage(defaultStorageConfig()), loaded: time.Now().UTC()})
	return m
}

func (m *ConfigManager) SetInvalidator(invalidator ConfigInvalidator) {
	if m != nil {
		m.invalidator = invalidator
	}
}

func (m *ConfigManager) Load(ctx context.Context) error {
	cfg, err := m.loadStorage(ctx)
	state := configState{storage: cfg, public: publicFromStorage(cfg), loaded: time.Now().UTC()}
	if err != nil {
		state.err = err.Error()
		m.state.Store(state)
		return err
	}
	active, activeErr := activeFromStorage(cfg, m.encryptor)
	if activeErr != nil {
		state.err = activeErr.Error()
		m.state.Store(state)
		return activeErr
	}
	state.active = active
	m.state.Store(state)
	return nil
}

func (m *ConfigManager) Active() (ActiveConfig, bool) {
	state := m.current()
	return state.active, state.active.Enabled && len(state.active.EnabledEndpoints()) > 0 && state.err == ""
}

func (m *ConfigManager) EffectiveMode() Mode {
	state := m.current()
	if state.err != "" || !state.active.Enabled {
		if state.storage.Enabled && state.storage.BlockingEnabled {
			return ModeBlocking
		}
		return ModeOff
	}
	return state.active.EffectiveMode()
}

func (m *ConfigManager) BlockingActivationDegraded() bool {
	state := m.current()
	return state.storage.Enabled && state.storage.BlockingEnabled && (state.err != "" || !state.active.Enabled)
}

func (m *ConfigManager) Public() PublicConfig {
	return m.current().public
}

func (m *ConfigManager) RuntimeState() (int64, int64, *time.Time, string) {
	state := m.current()
	loaded := state.loaded
	return state.storage.ConfigVersion, state.active.ConfigVersion, &loaded, state.err
}

func (m *ConfigManager) Save(ctx context.Context, req UpdateConfigRequest, actorID int64) (PublicConfig, error) {
	if err := validateUpdate(req); err != nil {
		return PublicConfig{}, err
	}
	current, err := m.loadStorage(ctx)
	if err != nil {
		return PublicConfig{}, err
	}
	if req.ExpectedConfigVersion > 0 && req.ExpectedConfigVersion != current.ConfigVersion {
		return PublicConfig{}, infraerrors.New(http.StatusConflict, ErrorCodeConfigConflict, "prompt audit config has changed")
	}
	next, err := m.buildStorageFromUpdate(req, current, actorID)
	if err != nil {
		return PublicConfig{}, err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return PublicConfig{}, err
	}
	if err := m.settings.Set(ctx, SettingKeyPromptAuditConfig, string(raw)); err != nil {
		return PublicConfig{}, err
	}
	if err := m.Load(ctx); err != nil {
		return PublicConfig{}, err
	}
	if m.invalidator != nil {
		if err := m.invalidator.Publish(ctx); err != nil {
			promptAuditLog(ctx).Warn("prompt_audit_config_invalidation_failed",
				zap.Int64("config_version", next.ConfigVersion),
				zap.String("error_code", "prompt_audit_config_invalidation_failed"),
			)
		} else {
			promptAuditLog(ctx).Info("prompt_audit_config_invalidation_published",
				zap.Int64("config_version", next.ConfigVersion),
			)
		}
	}
	return m.Public(), nil
}

func (m *ConfigManager) Encrypt(value string) (string, error) {
	if m == nil || m.encryptor == nil {
		return "", errors.New("prompt audit encryptor unavailable")
	}
	return m.encryptor.Encrypt(value)
}

func (m *ConfigManager) Decrypt(value string) (string, error) {
	if m == nil || m.encryptor == nil {
		return "", errors.New("prompt audit encryptor unavailable")
	}
	return m.encryptor.Decrypt(value)
}

func (m *ConfigManager) current() configState {
	state, _ := m.state.Load().(configState)
	return state
}

func (m *ConfigManager) loadStorage(ctx context.Context) (storageConfig, error) {
	cfg := defaultStorageConfig()
	if m == nil || m.settings == nil {
		return cfg, nil
	}
	raw, err := m.settings.GetValue(ctx, SettingKeyPromptAuditConfig)
	if err != nil {
		if errors.Is(err, service.ErrSettingNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	return parseStorageConfig(raw)
}

func (m *ConfigManager) buildStorageFromUpdate(req UpdateConfigRequest, current storageConfig, actorID int64) (storageConfig, error) {
	next := storageConfig{
		Enabled: req.Enabled, BlockingEnabled: req.BlockingEnabled, StorePassEvents: req.StorePassEvents,
		Strategy: strings.TrimSpace(req.Strategy), WorkerCount: req.WorkerCount, QueueCapacity: req.QueueCapacity,
		Scanners: canonicalScannerIDs(req.Scanners), AllGroups: req.AllGroups, GroupIDs: canonicalInt64s(req.GroupIDs),
		ConfigVersion: current.ConfigVersion + 1, UpdatedAt: time.Now().UTC(), UpdatedBy: actorID,
	}
	for _, input := range req.Endpoints {
		endpoint, err := m.buildEndpoint(input, current)
		if err != nil {
			return storageConfig{}, err
		}
		next.Endpoints = append(next.Endpoints, endpoint)
	}
	normalizeStorageConfig(&next)
	if err := validateStorageConfig(next); err != nil {
		return storageConfig{}, err
	}
	next.ChangeSummary = changeSummary(next)
	return next, nil
}
