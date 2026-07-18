package routes

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type promptAuditRouteAuditRepo struct {
	created []*service.AuditLog
}

func (r *promptAuditRouteAuditRepo) CreateAuditLog(_ context.Context, log *service.AuditLog) error {
	clone := *log
	r.created = append(r.created, &clone)
	return nil
}

func (r *promptAuditRouteAuditRepo) ListAuditLogs(context.Context, *service.AuditLogFilter) (*service.AuditLogList, error) {
	return &service.AuditLogList{}, nil
}

func (r *promptAuditRouteAuditRepo) DeleteAuditLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type promptAuditRouteScanner struct{}

func (s *promptAuditRouteScanner) Scan(context.Context, securityaudit.ActiveEndpoint, string, []string) (*securityaudit.NormalizedResult, error) {
	return promptAuditRoutePassResult(), nil
}

type promptAuditRouteSettings struct {
	values map[string]string
}

func (s *promptAuditRouteSettings) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (s *promptAuditRouteSettings) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *promptAuditRouteSettings) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *promptAuditRouteSettings) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, err := s.GetValue(ctx, key); err == nil {
			out[key] = value
		}
	}
	return out, nil
}

func (s *promptAuditRouteSettings) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := s.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *promptAuditRouteSettings) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *promptAuditRouteSettings) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type promptAuditRouteEncryptor struct{}

func (promptAuditRouteEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (promptAuditRouteEncryptor) Decrypt(value string) (string, error) { return value, nil }

func newPromptAuditRouteService(t *testing.T) *securityaudit.PromptService {
	t.Helper()
	manager := securityaudit.NewConfigManager(&promptAuditRouteSettings{values: map[string]string{}}, promptAuditRouteEncryptor{})
	_, err := manager.Save(context.Background(), promptAuditRouteConfig(), 42)
	require.NoError(t, err)
	return securityaudit.NewPromptService(
		manager,
		&promptAuditRouteRepo{},
		&promptAuditRoutePayloadStore{},
		&promptAuditRouteScanner{},
		securityaudit.NewAtomicMetrics(),
	)
}
