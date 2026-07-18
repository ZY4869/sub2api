package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type backupHandlerSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newBackupHandlerSettingRepo() *backupHandlerSettingRepo {
	return &backupHandlerSettingRepo{values: map[string]string{}}
}

func (r *backupHandlerSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *backupHandlerSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values[key], nil
}

func (r *backupHandlerSettingRepo) Set(_ context.Context, key string, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *backupHandlerSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, _ := r.GetValue(ctx, key); value != "" {
			out[key] = value
		}
	}
	return out, nil
}

func (r *backupHandlerSettingRepo) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := r.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *backupHandlerSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *backupHandlerSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

type backupHandlerPlainEncryptor struct{}

func (backupHandlerPlainEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (backupHandlerPlainEncryptor) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "enc:"), nil
}

func TestBackupHandlerUpdateS3ConfigRequiresStepUpTotp(t *testing.T) {
	handler, repo := newBackupHandlerForTest(false)
	router := gin.New()
	router.PUT("/admin/backups/s3-config", handler.UpdateS3Config)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/backups/s3-config", strings.NewReader(validBackupS3ConfigJSON()))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Empty(t, repo.values, "S3 config should not be saved without step-up")
}

func TestBackupHandlerUpdateS3ConfigAcceptsStepUpAndRedactsSecret(t *testing.T) {
	handler, repo := newBackupHandlerForTest(true)
	router := gin.New()
	router.PUT("/admin/backups/s3-config", handler.UpdateS3Config)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/backups/s3-config", strings.NewReader(validBackupS3ConfigJSON()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminStepUpTotpHeader, "123456")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, repo.values)
	require.NotContains(t, w.Body.String(), "super-secret")
}

func newBackupHandlerForTest(allowStepUp bool) (*BackupHandler, *backupHandlerSettingRepo) {
	gin.SetMode(gin.TestMode)
	repo := newBackupHandlerSettingRepo()
	cfg := &config.Config{Database: config.DatabaseConfig{Host: "localhost", Port: 5432, User: "test", DBName: "test"}}
	backupService := service.NewBackupService(repo, cfg, backupHandlerPlainEncryptor{}, nil, noopBackupDumper{})
	handler := NewBackupHandler(backupService, nil)
	handler.SetAdminSecurityHelper(&AdminSecurityHelper{stepUpVerifier: func(c *gin.Context, scope string) bool {
		if allowStepUp && strings.TrimSpace(c.GetHeader(adminStepUpTotpHeader)) != "" {
			return true
		}
		c.JSON(http.StatusForbidden, gin.H{"code": "STEP_UP_REQUIRED", "scope": scope})
		return false
	}})
	return handler, repo
}

func validBackupS3ConfigJSON() string {
	raw, _ := json.Marshal(service.BackupS3Config{
		Endpoint:        "https://s3.example.com",
		Region:          "auto",
		Bucket:          "bucket",
		AccessKeyID:     "ak",
		SecretAccessKey: "super-secret",
		Prefix:          "backups/",
	})
	return string(raw)
}

type noopBackupDumper struct{}

func (noopBackupDumper) Dump(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (noopBackupDumper) Restore(context.Context, io.Reader) error { return nil }
