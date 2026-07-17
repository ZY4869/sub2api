package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type ChannelMonitorService struct {
	db           *sql.DB
	repo         ChannelMonitorRepository
	templateRepo ChannelMonitorTemplateRepository
	historyRepo  ChannelMonitorHistoryRepository
	rollupRepo   ChannelMonitorRollupRepository
	settingSvc   *SettingService
	encryptor    SecretEncryptor
	cfg          *config.Config
	checker      *channelMonitorHTTPChecker
	accountRepo  AccountRepository
	testRunner   channelMonitorAccountTestRunner
}

func NewChannelMonitorService(
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
) *ChannelMonitorService {
	return &ChannelMonitorService{
		repo:        repo,
		historyRepo: historyRepo,
		rollupRepo:  rollupRepo,
		settingSvc:  settingSvc,
		encryptor:   encryptor,
		cfg:         cfg,
		checker:     newChannelMonitorHTTPChecker(cfg),
	}
}

func (s *ChannelMonitorService) SetTemplateRepository(db *sql.DB, templateRepo ChannelMonitorTemplateRepository) {
	s.db = db
	s.templateRepo = templateRepo
}

func (s *ChannelMonitorService) SetAccountMonitorDependencies(accountRepo AccountRepository, testRunner channelMonitorAccountTestRunner) {
	s.accountRepo = accountRepo
	s.testRunner = testRunner
}

func (s *ChannelMonitorService) ListAll(ctx context.Context) ([]*ChannelMonitor, error) {
	return s.repo.ListAll(ctx)
}

func (s *ChannelMonitorService) GetByID(ctx context.Context, id int64) (*ChannelMonitor, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ChannelMonitorService) Create(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string) (*ChannelMonitor, error) {
	prepared, err := s.prepareCreate(ctx, monitor, plaintextAPIKey)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, prepared)
}

type ChannelMonitorTemplateCreateInput struct {
	Save bool
	Name string
}

func (s *ChannelMonitorService) CreateWithOptionalTemplate(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string, templateInput ChannelMonitorTemplateCreateInput) (*ChannelMonitor, error) {
	if !templateInput.Save {
		return s.Create(ctx, monitor, plaintextAPIKey)
	}
	if s.db == nil || s.templateRepo == nil {
		return nil, ErrChannelMonitorInvalidRequest
	}
	monitorTxRepo, ok := s.repo.(ChannelMonitorRepositoryTxCreator)
	if !ok {
		return nil, ErrChannelMonitorInvalidRequest
	}
	templateTxRepo, ok := s.templateRepo.(ChannelMonitorTemplateRepositoryTxCreator)
	if !ok {
		return nil, ErrChannelMonitorInvalidRequest
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	prepared, err := s.prepareCreate(ctx, monitor, plaintextAPIKey)
	if err != nil {
		return nil, err
	}
	templateName := strings.TrimSpace(templateInput.Name)
	if templateName == "" {
		templateName = strings.TrimSpace(prepared.Name) + " 模板"
	}
	tpl, err := normalizeChannelMonitorTemplate(&ChannelMonitorRequestTemplate{
		Name:               templateName,
		Provider:           prepared.Provider,
		RequestProtocol:    prepared.RequestProtocol,
		ExtraHeaders:       prepared.ExtraHeaders,
		BodyOverrideMode:   prepared.BodyOverrideMode,
		BodyOverride:       prepared.BodyOverride,
		OpenAIAPIMode:      prepared.OpenAIAPIMode,
		TestPromptTemplate: prepared.TestPromptTemplate,
	})
	if err != nil {
		return nil, err
	}
	createdTpl, err := templateTxRepo.CreateWithTx(ctx, tx, tpl)
	if err != nil {
		return nil, err
	}
	prepared.TemplateID = &createdTpl.ID

	created, err := monitorTxRepo.CreateWithTx(ctx, tx, prepared)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return created, nil
}

type ChannelMonitorDuplicateInput struct {
	Name string
}

func (s *ChannelMonitorService) Duplicate(ctx context.Context, id int64, input *ChannelMonitorDuplicateInput) (*ChannelMonitor, error) {
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrChannelMonitorNotFound
	}
	name := ""
	if input != nil {
		name = strings.TrimSpace(input.Name)
	}
	if name == "" {
		name = duplicateChannelMonitorName(source.Name)
	}
	monitor := &ChannelMonitor{
		Name:                 name,
		Provider:             source.Provider,
		ProbeMode:            source.ProbeMode,
		RequestProtocol:      source.RequestProtocol,
		Endpoint:             source.Endpoint,
		APIKeyEncrypted:      cloneStringPtr(source.APIKeyEncrypted),
		IntervalSeconds:      source.IntervalSeconds,
		JitterSeconds:        source.JitterSeconds,
		Enabled:              false,
		AccountIDs:           append([]int64(nil), source.AccountIDs...),
		PrimaryModelID:       source.PrimaryModelID,
		AdditionalModelIDs:   append([]string(nil), source.AdditionalModelIDs...),
		ModelSourceProtocols: cloneChannelMonitorStringMap(source.ModelSourceProtocols),
		ModelProbeStrategy:   source.ModelProbeStrategy,
		TestPromptTemplate:   source.TestPromptTemplate,
		TemplateID:           cloneInt64Ptr(source.TemplateID),
		ExtraHeaders:         cloneChannelMonitorStringMap(source.ExtraHeaders),
		BodyOverrideMode:     source.BodyOverrideMode,
		BodyOverride:         cloneJSONMap(source.BodyOverride),
		OpenAIAPIMode:        source.OpenAIAPIMode,
	}
	return s.Create(ctx, monitor, nil)
}

func duplicateChannelMonitorName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Channel Monitor"
	}
	const suffix = " 副本"
	limit := 100 - len([]rune(suffix))
	runes := []rune(base)
	if limit < 1 {
		limit = 1
	}
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes) + suffix
}

func cloneChannelMonitorStringMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	out := make(map[string]string, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func (s *ChannelMonitorService) Update(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string) (*ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}
	existing, err := s.repo.GetByID(ctx, monitor.ID)
	if err != nil {
		return nil, err
	}

	merged := *existing
	merged.Name = monitor.Name
	merged.Provider = monitor.Provider
	merged.ProbeMode = monitor.ProbeMode
	merged.RequestProtocol = monitor.RequestProtocol
	merged.Endpoint = monitor.Endpoint
	merged.IntervalSeconds = monitor.IntervalSeconds
	merged.Enabled = monitor.Enabled
	merged.AccountIDs = monitor.AccountIDs
	merged.PrimaryModelID = monitor.PrimaryModelID
	merged.AdditionalModelIDs = monitor.AdditionalModelIDs
	merged.ModelSourceProtocols = monitor.ModelSourceProtocols
	merged.ModelProbeStrategy = monitor.ModelProbeStrategy
	merged.TestPromptTemplate = monitor.TestPromptTemplate
	merged.TemplateID = monitor.TemplateID
	merged.ExtraHeaders = monitor.ExtraHeaders
	merged.BodyOverrideMode = monitor.BodyOverrideMode
	merged.BodyOverride = monitor.BodyOverride
	merged.OpenAIAPIMode = monitor.OpenAIAPIMode

	runtime, err := s.settingSvc.GetChannelMonitorRuntime(ctx)
	defaultInterval := 60
	if err == nil && runtime != nil && runtime.DefaultIntervalSeconds > 0 {
		defaultInterval = runtime.DefaultIntervalSeconds
	}
	if merged.IntervalSeconds <= 0 {
		merged.IntervalSeconds = defaultInterval
	}

	normalized, err := normalizeChannelMonitor(&merged)
	if err != nil {
		return nil, err
	}
	merged = *normalized

	if merged.ProbeMode == ChannelMonitorProbeModeDirect {
		endpoint, err := validateChannelMonitorEndpointForSave(s.cfg, merged.Endpoint, merged.Enabled)
		if err != nil {
			return nil, err
		}
		merged.Endpoint = endpoint
	} else {
		merged.Endpoint = ""
		merged.APIKeyEncrypted = nil
	}

	if plaintextAPIKey != nil {
		key := strings.TrimSpace(*plaintextAPIKey)
		if key == "" {
			merged.APIKeyEncrypted = nil
		} else {
			enc, err := s.encryptor.Encrypt(key)
			if err != nil {
				return nil, err
			}
			merged.APIKeyEncrypted = &enc
		}
	}

	if merged.ProbeMode == ChannelMonitorProbeModeDirect && merged.Enabled && (merged.APIKeyEncrypted == nil || strings.TrimSpace(*merged.APIKeyEncrypted) == "") {
		return nil, ErrChannelMonitorAPIKeyRequired
	}

	ensureNextRunAtOnEnable(&merged, time.Now())
	return s.repo.Update(ctx, &merged)
}

func (s *ChannelMonitorService) prepareCreate(ctx context.Context, monitor *ChannelMonitor, plaintextAPIKey *string) (*ChannelMonitor, error) {
	if monitor == nil {
		return nil, errors.New("nil monitor")
	}

	runtime, err := s.settingSvc.GetChannelMonitorRuntime(ctx)
	defaultInterval := 60
	if err == nil && runtime != nil && runtime.DefaultIntervalSeconds > 0 {
		defaultInterval = runtime.DefaultIntervalSeconds
	}
	if monitor.IntervalSeconds <= 0 {
		monitor.IntervalSeconds = defaultInterval
	}

	normalized, err := normalizeChannelMonitor(monitor)
	if err != nil {
		return nil, err
	}
	prepared := *normalized

	if prepared.ProbeMode == ChannelMonitorProbeModeDirect {
		endpoint, err := validateChannelMonitorEndpointForSave(s.cfg, prepared.Endpoint, prepared.Enabled)
		if err != nil {
			return nil, err
		}
		prepared.Endpoint = endpoint
	} else {
		prepared.Endpoint = ""
	}

	if plaintextAPIKey != nil {
		key := strings.TrimSpace(*plaintextAPIKey)
		if key != "" {
			enc, err := s.encryptor.Encrypt(key)
			if err != nil {
				return nil, err
			}
			prepared.APIKeyEncrypted = &enc
		}
	}
	if prepared.ProbeMode == ChannelMonitorProbeModeDirect && prepared.Enabled && (prepared.APIKeyEncrypted == nil || strings.TrimSpace(*prepared.APIKeyEncrypted) == "") {
		return nil, ErrChannelMonitorAPIKeyRequired
	}

	ensureNextRunAtOnEnable(&prepared, time.Now())
	return &prepared, nil
}

func (s *ChannelMonitorService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *ChannelMonitorService) IsAPIKeyDecryptFailed(monitor *ChannelMonitor) bool {
	if monitor == nil || monitor.APIKeyEncrypted == nil || strings.TrimSpace(*monitor.APIKeyEncrypted) == "" {
		return false
	}
	if s.encryptor == nil {
		return true
	}
	_, err := s.encryptor.Decrypt(*monitor.APIKeyEncrypted)
	return err != nil
}

func (s *ChannelMonitorService) RunCheckNow(ctx context.Context, monitorID int64) ([]*ChannelMonitorHistory, error) {
	monitor, err := s.repo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	last := now
	next := now.Add(time.Duration(monitor.IntervalSeconds) * time.Second)
	monitor.LastRunAt = &last
	monitor.NextRunAt = &next
	if _, err := s.repo.Update(ctx, monitor); err != nil {
		return nil, err
	}

	exec := newChannelMonitorExecutor(s.encryptor, s.cfg, s.checker, s.historyRepo)
	exec.accountRepo = s.accountRepo
	exec.testRunner = s.testRunner
	return exec.Execute(ctx, monitor)
}
