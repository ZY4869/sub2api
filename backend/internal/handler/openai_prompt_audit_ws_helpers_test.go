package handler

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type wsPromptAuditScanner struct {
	result *securityaudit.NormalizedResult
	err    error
}

func (s *wsPromptAuditScanner) Scan(context.Context, securityaudit.ActiveEndpoint, string, []string) (*securityaudit.NormalizedResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type wsPromptAuditSequenceScanner struct {
	results []*securityaudit.NormalizedResult
	errs    []error
	calls   int
	chunks  []string
}

func (s *wsPromptAuditSequenceScanner) Scan(_ context.Context, _ securityaudit.ActiveEndpoint, chunk string, _ []string) (*securityaudit.NormalizedResult, error) {
	s.calls++
	s.chunks = append(s.chunks, chunk)
	index := s.calls - 1
	if index < len(s.errs) && s.errs[index] != nil {
		return nil, s.errs[index]
	}
	if index < len(s.results) {
		return s.results[index], nil
	}
	return nil, nil
}

type wsPromptAuditSettings struct {
	values map[string]string
}

func (s *wsPromptAuditSettings) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (s *wsPromptAuditSettings) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *wsPromptAuditSettings) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *wsPromptAuditSettings) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, err := s.GetValue(ctx, key); err == nil {
			out[key] = value
		}
	}
	return out, nil
}

func (s *wsPromptAuditSettings) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := s.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *wsPromptAuditSettings) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *wsPromptAuditSettings) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type wsPromptAuditEncryptor struct{}

func (wsPromptAuditEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (wsPromptAuditEncryptor) Decrypt(value string) (string, error) {
	if len(value) < 4 || value[:4] != "enc:" {
		return "", stderrors.New("ciphertext invalid")
	}
	return value[4:], nil
}

type wsPromptAuditRepo struct{}

func (r *wsPromptAuditRepo) CreateJob(context.Context, *securityaudit.Job) error { return nil }
func (r *wsPromptAuditRepo) CreateEvent(context.Context, *securityaudit.Job, *securityaudit.NormalizedResult, string) (*securityaudit.Event, error) {
	return &securityaudit.Event{ID: 1}, nil
}
func (r *wsPromptAuditRepo) ClaimJobs(context.Context, int) ([]*securityaudit.Job, error) {
	return nil, nil
}
func (r *wsPromptAuditRepo) MarkJobDone(context.Context, int64) error { return nil }
func (r *wsPromptAuditRepo) MarkJobFailed(context.Context, *securityaudit.Job, string, string, bool) error {
	return nil
}
func (r *wsPromptAuditRepo) QueueStats(context.Context) (securityaudit.QueueStats, error) {
	return securityaudit.QueueStats{}, nil
}
func (r *wsPromptAuditRepo) ListEvents(context.Context, securityaudit.EventFilter) (*securityaudit.EventList, error) {
	return &securityaudit.EventList{}, nil
}
func (r *wsPromptAuditRepo) GetEvent(context.Context, int64, bool) (*securityaudit.Event, error) {
	return nil, securityaudit.ErrEventNotFound
}
func (r *wsPromptAuditRepo) DeleteEvent(context.Context, int64) (*securityaudit.DeleteResult, error) {
	return &securityaudit.DeleteResult{}, nil
}
func (r *wsPromptAuditRepo) PreviewDelete(context.Context, securityaudit.EventFilter) (*securityaudit.DeletePreview, error) {
	return &securityaudit.DeletePreview{}, nil
}
func (r *wsPromptAuditRepo) DeleteEventsByFilter(context.Context, securityaudit.EventFilter, int64) (*securityaudit.DeleteResult, error) {
	return &securityaudit.DeleteResult{}, nil
}

type wsPromptAuditPayloadStore struct{}

func (s *wsPromptAuditPayloadStore) Set(context.Context, int64, securityaudit.PromptPayload, time.Duration) error {
	return nil
}
func (s *wsPromptAuditPayloadStore) Get(context.Context, int64) (securityaudit.PromptPayload, error) {
	return securityaudit.PromptPayload{}, stderrors.New("missing payload")
}
func (s *wsPromptAuditPayloadStore) Delete(context.Context, int64) error { return nil }
func (s *wsPromptAuditPayloadStore) Ping(context.Context) error          { return nil }

func newWSPromptAuditService(t *testing.T, scanner securityaudit.PromptScanner) *securityaudit.PromptService {
	t.Helper()
	manager := securityaudit.NewConfigManager(&wsPromptAuditSettings{values: map[string]string{}}, wsPromptAuditEncryptor{})
	_, err := manager.Save(context.Background(), wsPromptAuditConfig(true), 7)
	require.NoError(t, err)
	return securityaudit.NewPromptService(manager, &wsPromptAuditRepo{}, &wsPromptAuditPayloadStore{}, scanner, securityaudit.NewAtomicMetrics())
}

func wsPromptAuditConfig(blocking bool) securityaudit.UpdateConfigRequest {
	return securityaudit.UpdateConfigRequest{
		Enabled: true, BlockingEnabled: blocking, StorePassEvents: true,
		Strategy: securityaudit.DefaultStrategy, WorkerCount: 1, QueueCapacity: 8,
		Scanners: securityaudit.AllScannerIDs, AllGroups: true,
		Endpoints: []securityaudit.UpdateEndpoint{{
			ID: "primary", Name: "Primary", Protocol: "openai_compatible",
			BaseURL: "https://guard.example.com/v1", Model: securityaudit.DefaultGuardModel,
			TimeoutMS: securityaudit.DefaultTimeoutMS, InputLimit: securityaudit.DefaultInputLimit, Enabled: true,
		}},
	}
}

func wsPromptAuditContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	ctx := context.WithValue(req.Context(), ctxkey.RequestID, "req-ws")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "corr-ws")
	c.Request = req.WithContext(ctx)
	return c
}

func wsPromptAuditAPIKey() *service.APIKey {
	groupID := int64(1)
	userID := int64(2)
	return &service.APIKey{
		ID: 3, Name: "key", UserID: userID, GroupID: &groupID,
		Group: &service.Group{ID: groupID, Name: "OpenAI", Platform: service.PlatformOpenAI, Hydrated: true},
		User:  &service.User{ID: userID, Username: "alice", Email: "alice@example.com"},
	}
}
