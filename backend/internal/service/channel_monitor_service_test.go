package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type channelMonitorServiceRepoStub struct {
	ChannelMonitorRepository
	createErr error
	getErr    error
	updateErr error
	created   *ChannelMonitor
	byID      *ChannelMonitor
	updated   *ChannelMonitor
	enabled   []*ChannelMonitor
}

func (s *channelMonitorServiceRepoStub) Create(_ context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	clone := *monitor
	clone.ID = 42
	clone.CreatedAt = time.Now().UTC()
	clone.UpdatedAt = clone.CreatedAt
	s.created = &clone
	return &clone, nil
}

func (s *channelMonitorServiceRepoStub) GetByID(_ context.Context, id int64) (*ChannelMonitor, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.byID == nil || s.byID.ID != id {
		return nil, ErrChannelMonitorNotFound
	}
	clone := *s.byID
	return &clone, nil
}

func (s *channelMonitorServiceRepoStub) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	return s.enabled, nil
}

func (s *channelMonitorServiceRepoStub) Update(_ context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	clone := *monitor
	s.updated = &clone
	s.byID = &clone
	return &clone, nil
}

type channelMonitorHistoryRepoStub struct {
	ChannelMonitorHistoryRepository
	latest  []*ChannelMonitorHistory
	created []*ChannelMonitorHistory
}

func (s *channelMonitorHistoryRepoStub) ListLatestByMonitorIDs(context.Context, []int64) ([]*ChannelMonitorHistory, error) {
	return s.latest, nil
}

func (s *channelMonitorHistoryRepoStub) Create(_ context.Context, history *ChannelMonitorHistory) (*ChannelMonitorHistory, error) {
	clone := *history
	clone.ID = int64(len(s.created) + 1)
	s.created = append(s.created, &clone)
	return &clone, nil
}

type channelMonitorAccountRepoStub struct {
	AccountRepository
	accounts []*Account
}

func (s *channelMonitorAccountRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	allowed := map[int64]struct{}{}
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	var out []*Account
	for _, account := range s.accounts {
		if account == nil {
			continue
		}
		if _, ok := allowed[account.ID]; ok {
			out = append(out, account)
		}
	}
	return out, nil
}

type channelMonitorAccountTestRunnerStub struct {
	calls  []ScheduledTestExecutionInput
	result *BackgroundAccountTestResult
}

func (s *channelMonitorAccountTestRunnerStub) RunTestBackgroundDetailed(_ context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls = append(s.calls, input)
	if s.result != nil {
		return s.result, nil
	}
	return &BackgroundAccountTestResult{
		Status:       "success",
		ResponseText: challengeFromPrompt(input.Prompt),
		LatencyMs:    120,
		StartedAt:    time.Now().Add(-120 * time.Millisecond),
		FinishedAt:   time.Now(),
	}, nil
}

type channelMonitorRollupRepoStub struct {
	ChannelMonitorRollupRepository
	daily []*ChannelMonitorDailyRollup
}

func (s *channelMonitorRollupRepoStub) ListDailyByMonitorIDs(context.Context, []int64, time.Time) ([]*ChannelMonitorDailyRollup, error) {
	return s.daily, nil
}

type channelMonitorTxRepoStub struct {
	ChannelMonitorRepository
	createErr error
	created   *ChannelMonitor
}

func (s *channelMonitorTxRepoStub) Create(ctx context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error) {
	return s.CreateWithTx(ctx, nil, monitor)
}

func (s *channelMonitorTxRepoStub) CreateWithTx(_ context.Context, _ *sql.Tx, monitor *ChannelMonitor) (*ChannelMonitor, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	clone := *monitor
	clone.ID = 801
	s.created = &clone
	return &clone, nil
}

type channelMonitorTemplateTxRepoStub struct {
	ChannelMonitorTemplateRepository
	created *ChannelMonitorRequestTemplate
}

func (s *channelMonitorTemplateTxRepoStub) Create(ctx context.Context, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error) {
	return s.CreateWithTx(ctx, nil, tpl)
}

func (s *channelMonitorTemplateTxRepoStub) CreateWithTx(_ context.Context, _ *sql.Tx, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error) {
	clone := *tpl
	clone.ID = 901
	s.created = &clone
	return &clone, nil
}

type channelMonitorEncryptorStub struct {
	encrypted string
}

func (s channelMonitorEncryptorStub) Encrypt(plaintext string) (string, error) {
	return s.encrypted + plaintext, nil
}

func (s channelMonitorEncryptorStub) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

func TestChannelMonitorService_Create_DisabledAllowsEmptyAPIKey(t *testing.T) {
	repo := &channelMonitorServiceRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)

	created, err := svc.Create(context.Background(), validChannelMonitorForCreate(false), nil)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Nil(t, created.APIKeyEncrypted)
	require.Equal(t, 60, created.IntervalSeconds)
	require.False(t, created.Enabled)
	require.Nil(t, created.NextRunAt)
	require.NotNil(t, repo.created)
}

func TestChannelMonitorService_Create_EnabledRequiresAPIKey(t *testing.T) {
	svc := newChannelMonitorServiceForCreateTest(&channelMonitorServiceRepoStub{})

	_, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), nil)
	require.ErrorIs(t, err, ErrChannelMonitorAPIKeyRequired)

	blank := "   "
	_, err = svc.Create(context.Background(), validChannelMonitorForCreate(true), &blank)
	require.ErrorIs(t, err, ErrChannelMonitorAPIKeyRequired)
}

func TestChannelMonitorService_Create_EncryptsAPIKeyAndSchedulesEnabledMonitor(t *testing.T) {
	repo := &channelMonitorServiceRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)
	key := "sk-secret"

	created, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), &key)
	require.NoError(t, err)
	require.NotNil(t, created.APIKeyEncrypted)
	require.Equal(t, "encrypted:sk-secret", *created.APIKeyEncrypted)
	require.NotNil(t, created.NextRunAt)
	require.NotContains(t, created.Endpoint, "sk-secret")
}

func TestChannelMonitorService_Create_AccountModeDefaultsPrimaryOnlyAndKeepsModelProtocols(t *testing.T) {
	repo := &channelMonitorServiceRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)

	created, err := svc.Create(context.Background(), &ChannelMonitor{
		Name:               "Pool health",
		Provider:           ChannelMonitorProviderOpenAI,
		ProbeMode:          ChannelMonitorProbeModeAccountPool,
		RequestProtocol:    ChannelMonitorRequestProtocolOpenAI,
		Enabled:            true,
		AccountIDs:         []int64{12, 11, 12},
		PrimaryModelID:     "shared-main",
		AdditionalModelIDs: []string{"shared-side"},
		ModelSourceProtocols: map[string]string{
			"shared-main":  ChannelMonitorRequestProtocolAnthropic,
			"shared-side":  ChannelMonitorRequestProtocolGemini,
			"not-selected": ChannelMonitorRequestProtocolOpenAI,
		},
		BodyOverrideMode: ChannelMonitorBodyOverrideModeOff,
	}, nil)

	require.NoError(t, err)
	require.Equal(t, ChannelMonitorModelProbeStrategyPrimaryOnly, created.ModelProbeStrategy)
	require.Equal(t, []int64{11, 12}, created.AccountIDs)
	require.Equal(t, map[string]string{
		"shared-main": ChannelMonitorRequestProtocolAnthropic,
		"shared-side": ChannelMonitorRequestProtocolGemini,
	}, created.ModelSourceProtocols)
	require.Empty(t, created.Endpoint)
	require.Nil(t, created.APIKeyEncrypted)
}

func TestChannelMonitorService_CreateWithOptionalTemplate_CommitsMonitorAndTemplateTogether(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	mock.ExpectCommit()

	monitorRepo := &channelMonitorTxRepoStub{}
	templateRepo := &channelMonitorTemplateTxRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(monitorRepo)
	svc.SetTemplateRepository(db, templateRepo)
	key := "sk-secret"

	created, err := svc.CreateWithOptionalTemplate(
		context.Background(),
		validChannelMonitorForCreate(true),
		&key,
		ChannelMonitorTemplateCreateInput{Save: true, Name: ""},
	)

	require.NoError(t, err)
	require.Equal(t, int64(801), created.ID)
	require.NotNil(t, created.TemplateID)
	require.Equal(t, int64(901), *created.TemplateID)
	require.NotNil(t, templateRepo.created)
	require.Equal(t, "OpenAI health 模板", templateRepo.created.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelMonitorService_CreateWithOptionalTemplate_RollsBackWhenMonitorCreateFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	mock.ExpectRollback()

	monitorRepo := &channelMonitorTxRepoStub{createErr: errors.New("monitor insert failed")}
	templateRepo := &channelMonitorTemplateTxRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(monitorRepo)
	svc.SetTemplateRepository(db, templateRepo)
	key := "sk-secret"

	_, err = svc.CreateWithOptionalTemplate(
		context.Background(),
		validChannelMonitorForCreate(true),
		&key,
		ChannelMonitorTemplateCreateInput{Save: true, Name: "Custom template"},
	)

	require.ErrorContains(t, err, "monitor insert failed")
	require.NotNil(t, templateRepo.created)
	require.Nil(t, monitorRepo.created)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProvideChannelMonitorService_WiresTemplateRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	mock.ExpectCommit()

	monitorRepo := &channelMonitorTxRepoStub{}
	templateRepo := &channelMonitorTemplateTxRepoStub{}
	settingRepo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyChannelMonitorDefaultIntervalSeconds: "60",
	}}
	svc := ProvideChannelMonitorService(
		db,
		monitorRepo,
		&channelMonitorHistoryRepoStub{},
		&channelMonitorRollupRepoStub{},
		NewSettingService(settingRepo, &config.Config{}),
		channelMonitorEncryptorStub{encrypted: "encrypted:"},
		&config.Config{},
		templateRepo,
	)
	key := "sk-secret"

	created, err := svc.CreateWithOptionalTemplate(
		context.Background(),
		validChannelMonitorForCreate(true),
		&key,
		ChannelMonitorTemplateCreateInput{Save: true},
	)

	require.NoError(t, err)
	require.NotNil(t, created.TemplateID)
	require.Equal(t, int64(901), *created.TemplateID)
	require.NotNil(t, templateRepo.created)
	require.Equal(t, "OpenAI health 模板", templateRepo.created.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelMonitorService_Create_ReturnsRepositoryError(t *testing.T) {
	svc := newChannelMonitorServiceForCreateTest(&channelMonitorServiceRepoStub{
		createErr: errors.New("db down"),
	})
	key := "sk-secret"

	_, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), &key)
	require.ErrorContains(t, err, "db down")
}

func TestChannelMonitorService_PublicModelCatalogHealthAggregatesAndSanitizes(t *testing.T) {
	now := time.Now().UTC()
	repo := &channelMonitorServiceRepoStub{
		enabled: []*ChannelMonitor{
			{ID: 10, Enabled: true, PrimaryModelID: "gpt-5.4"},
			{ID: 11, Enabled: true, PrimaryModelID: "gpt-5.4"},
		},
	}
	historyRepo := &channelMonitorHistoryRepoStub{
		latest: []*ChannelMonitorHistory{
			{MonitorID: 10, ModelID: "gpt-5.4", Status: ChannelMonitorStatusSuccess, LatencyMs: 320, CreatedAt: now.Add(-time.Minute)},
			{MonitorID: 11, ModelID: "gpt-5.4-source", Status: ChannelMonitorStatusDegraded, LatencyMs: 640, CreatedAt: now},
		},
	}
	rollupRepo := &channelMonitorRollupRepoStub{
		daily: []*ChannelMonitorDailyRollup{
			{MonitorID: 10, ModelID: "gpt-5.4", Day: now, TotalChecks: 10, AvailableChecks: 10, TotalLatencyMs: 3200},
			{MonitorID: 11, ModelID: "gpt-5.4-source", Day: now, TotalChecks: 10, AvailableChecks: 8, DegradedChecks: 2, TotalLatencyMs: 5120},
		},
	}
	svc := NewChannelMonitorService(repo, historyRepo, rollupRepo, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{
		Model:     "gpt-5.4-public",
		SourceIDs: []string{"gpt-5.4", "gpt-5.4-source"},
	}})

	require.NoError(t, err)
	status := statuses["gpt-5.4-public"]
	require.Equal(t, PublicModelHealthStatusWarning, status.Status)
	require.Equal(t, "gpt-5.4-public", status.Model)
	require.NotNil(t, status.SuccessRateToday)
	require.InDelta(t, 0.9, *status.SuccessRateToday, 0.0001)
	require.NotNil(t, status.LatencyMs)
	require.Equal(t, int64(640), *status.LatencyMs)
	require.NotEmpty(t, status.Daily)
	require.NotContains(t, mustModelCatalogJSON(t, status), "monitor_id")
	require.NotContains(t, mustModelCatalogJSON(t, status), "gpt-5.4-source")
}

func TestChannelMonitorService_PublicModelCatalogHealthPendingWithoutMonitorData(t *testing.T) {
	svc := NewChannelMonitorService(&channelMonitorServiceRepoStub{}, &channelMonitorHistoryRepoStub{}, &channelMonitorRollupRepoStub{}, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{Model: "gpt-5.4"}})

	require.NoError(t, err)
	require.Equal(t, PublicModelHealthStatusPending, statuses["gpt-5.4"].Status)
	require.Empty(t, statuses["gpt-5.4"].Daily)
}

func TestChannelMonitorService_PublicModelCatalogHealthStaleHistoryHidesMetrics(t *testing.T) {
	stale := time.Now().UTC().Add(-(publicModelCatalogProbeHistoryTTL + time.Minute))
	repo := &channelMonitorServiceRepoStub{
		enabled: []*ChannelMonitor{{ID: 10, Enabled: true, PrimaryModelID: "gpt-5.4"}},
	}
	historyRepo := &channelMonitorHistoryRepoStub{
		latest: []*ChannelMonitorHistory{
			{MonitorID: 10, ModelID: "gpt-5.4", Status: ChannelMonitorStatusSuccess, LatencyMs: 320, CreatedAt: stale},
		},
	}
	rollupRepo := &channelMonitorRollupRepoStub{
		daily: []*ChannelMonitorDailyRollup{
			{MonitorID: 10, ModelID: "gpt-5.4", Day: stale, TotalChecks: 10, AvailableChecks: 10, TotalLatencyMs: 3200},
		},
	}
	svc := NewChannelMonitorService(repo, historyRepo, rollupRepo, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{Model: "gpt-5.4"}})

	require.NoError(t, err)
	status := statuses["gpt-5.4"]
	require.Equal(t, PublicModelHealthStatusPending, status.Status)
	require.Equal(t, PublicModelHealthSourceNone, status.HealthSource)
	require.Equal(t, PublicModelHealthReasonStaleHistory, status.StatusReason)
	require.NotEmpty(t, status.LastCheckedAt)
	require.Nil(t, status.SuccessRateToday)
	require.Nil(t, status.SuccessRate7d)
	require.Nil(t, status.LatencyMs)
	require.Empty(t, status.Daily)
	require.Empty(t, status.Trend)
}

func TestChannelMonitorExecutor_AccountPoolPrimaryOnlyUsesModelProtocolSnapshot(t *testing.T) {
	historyRepo := &channelMonitorHistoryRepoStub{}
	runner := &channelMonitorAccountTestRunnerStub{}
	exec := newChannelMonitorExecutor(nil, &config.Config{}, newChannelMonitorHTTPChecker(&config.Config{}), historyRepo)
	exec.accountRepo = &channelMonitorAccountRepoStub{accounts: []*Account{{ID: 101, Name: "alpha"}}}
	exec.testRunner = runner

	histories, err := exec.Execute(context.Background(), &ChannelMonitor{
		ID:                 9,
		ProbeMode:          ChannelMonitorProbeModeAccountPool,
		RequestProtocol:    ChannelMonitorRequestProtocolOpenAI,
		AccountIDs:         []int64{101},
		PrimaryModelID:     "claude-sonnet",
		AdditionalModelIDs: []string{"gemini-pro"},
		ModelProbeStrategy: ChannelMonitorModelProbeStrategyPrimaryOnly,
		ModelSourceProtocols: map[string]string{
			"claude-sonnet": ChannelMonitorRequestProtocolAnthropic,
			"gemini-pro":    ChannelMonitorRequestProtocolGemini,
		},
		TestPromptTemplate: "只回复 {{challenge}}",
	})

	require.NoError(t, err)
	require.Len(t, runner.calls, 1)
	require.Equal(t, int64(101), runner.calls[0].AccountID)
	require.Equal(t, "claude-sonnet", runner.calls[0].ModelID)
	require.Equal(t, ChannelMonitorRequestProtocolAnthropic, runner.calls[0].SourceProtocol)
	require.Contains(t, runner.calls[0].Prompt, "只回复 ")
	require.Len(t, histories, 1)
	require.Equal(t, ChannelMonitorStatusSuccess, histories[0].Status)
	require.Equal(t, "alpha", histories[0].AccountNameSnapshot)
	require.Len(t, historyRepo.created, 1)
}

func TestChannelMonitorExecutor_AccountPoolAllSelectedWritesHistoryPerAccountAndModel(t *testing.T) {
	historyRepo := &channelMonitorHistoryRepoStub{}
	runner := &channelMonitorAccountTestRunnerStub{}
	exec := newChannelMonitorExecutor(nil, &config.Config{}, newChannelMonitorHTTPChecker(&config.Config{}), historyRepo)
	exec.testRunner = runner

	histories, err := exec.Execute(context.Background(), &ChannelMonitor{
		ID:                 10,
		ProbeMode:          ChannelMonitorProbeModeAccountPool,
		RequestProtocol:    ChannelMonitorRequestProtocolOpenAI,
		AccountIDs:         []int64{201, 202},
		PrimaryModelID:     "main",
		AdditionalModelIDs: []string{"side"},
		ModelProbeStrategy: ChannelMonitorModelProbeStrategyAllSelected,
	})

	require.NoError(t, err)
	require.Len(t, runner.calls, 4)
	require.Len(t, histories, 4)
	require.Len(t, historyRepo.created, 4)
	require.Equal(t, "main", runner.calls[0].ModelID)
	require.Equal(t, "side", runner.calls[1].ModelID)
	require.Equal(t, ChannelMonitorProbeModeAccountPool, histories[0].ProbeMode)
	require.NotNil(t, histories[0].AccountID)
	require.Equal(t, int64(201), *histories[0].AccountID)
}

func TestChannelMonitorService_RunCheckNow_UsesBoundAccountRunner(t *testing.T) {
	historyRepo := &channelMonitorHistoryRepoStub{}
	repo := &channelMonitorServiceRepoStub{
		byID: &ChannelMonitor{
			ID:                 77,
			ProbeMode:          ChannelMonitorProbeModeAccountPool,
			RequestProtocol:    ChannelMonitorRequestProtocolOpenAI,
			IntervalSeconds:    60,
			Enabled:            true,
			AccountIDs:         []int64{101},
			PrimaryModelID:     "main",
			ModelProbeStrategy: ChannelMonitorModelProbeStrategyPrimaryOnly,
		},
	}
	runner := &channelMonitorAccountTestRunnerStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)
	svc.historyRepo = historyRepo
	_ = BindChannelMonitorAccountDependencies(
		svc,
		&channelMonitorAccountRepoStub{accounts: []*Account{{ID: 101, Name: "alpha"}}},
		runner,
	)

	histories, err := svc.RunCheckNow(context.Background(), 77)

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.NotNil(t, repo.updated.LastRunAt)
	require.NotNil(t, repo.updated.NextRunAt)
	require.Len(t, runner.calls, 1)
	require.Equal(t, int64(101), runner.calls[0].AccountID)
	require.Equal(t, "main", runner.calls[0].ModelID)
	require.Len(t, histories, 1)
	require.Len(t, historyRepo.created, 1)
	require.Equal(t, ChannelMonitorStatusSuccess, histories[0].Status)
	require.Equal(t, "alpha", histories[0].AccountNameSnapshot)
}

func TestBuildChannelMonitorPrompt_AppendsChallengeWhenTemplateHasNoPlaceholder(t *testing.T) {
	prompt := buildChannelMonitorPrompt("请简短回答", "abc123")

	require.Contains(t, prompt, "请简短回答")
	require.Contains(t, prompt, "abc123")
}

func newChannelMonitorServiceForCreateTest(repo ChannelMonitorRepository) *ChannelMonitorService {
	settingRepo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyChannelMonitorDefaultIntervalSeconds: "60",
	}}
	return NewChannelMonitorService(
		repo,
		nil,
		nil,
		NewSettingService(settingRepo, &config.Config{}),
		channelMonitorEncryptorStub{encrypted: "encrypted:"},
		&config.Config{},
	)
}

func validChannelMonitorForCreate(enabled bool) *ChannelMonitor {
	return &ChannelMonitor{
		Name:             "OpenAI health",
		Provider:         ChannelMonitorProviderOpenAI,
		Endpoint:         "https://api.openai.example/v1/responses",
		Enabled:          enabled,
		PrimaryModelID:   "gpt-5.4",
		BodyOverrideMode: ChannelMonitorBodyOverrideModeOff,
	}
}

func challengeFromPrompt(prompt string) string {
	marker := "exactly: "
	if idx := strings.LastIndex(prompt, marker); idx >= 0 {
		return strings.TrimSpace(prompt[idx+len(marker):])
	}
	fields := strings.Fields(prompt)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
