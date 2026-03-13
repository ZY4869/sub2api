//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type telegramNotifierRepoStub struct {
	values map[string]string
}

func (s *telegramNotifierRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *telegramNotifierRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *telegramNotifierRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *telegramNotifierRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *telegramNotifierRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *telegramNotifierRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}

func (s *telegramNotifierRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestTelegramNotifierService_SendNotification(t *testing.T) {
	var gotPath string
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	repo := &telegramNotifierRepoStub{
		values: map[string]string{
			SettingKeyTelegramBotToken: "123456:ABCDEF-token",
			SettingKeyTelegramChatID:   "-100123456",
		},
	}
	svc := NewSettingService(repo, &config.Config{})
	notifier := NewTelegramNotifierService(svc)
	notifier.apiBaseURL = server.URL
	notifier.client = server.Client()

	err := notifier.SendNotification(context.Background(), "hello telegram")
	require.NoError(t, err)
	require.Equal(t, "/bot123456:ABCDEF-token/sendMessage", gotPath)
	require.Contains(t, gotBody, `"chat_id":"-100123456"`)
	require.Contains(t, gotBody, `"text":"hello telegram"`)
}

func TestTelegramNotifierService_TestConnectionReturnsTelegramError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"too many requests"}`))
	}))
	defer server.Close()

	notifier := NewTelegramNotifierService(NewSettingService(&telegramNotifierRepoStub{}, &config.Config{}))
	notifier.apiBaseURL = server.URL
	notifier.client = server.Client()

	err := notifier.TestConnection(context.Background(), "123456:ABCDEF-token", "-100123456")
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many requests")
}

func TestTelegramNotifierService_RateLimit(t *testing.T) {
	now := time.Now()
	notifier := NewTelegramNotifierService(NewSettingService(&telegramNotifierRepoStub{}, &config.Config{}))
	notifier.now = func() time.Time { return now }
	for i := 0; i < telegramSendLimitPerMinute; i++ {
		notifier.recentSends = append(notifier.recentSends, now.Add(-time.Second))
	}

	err := notifier.sendMessage(context.Background(), "123456:ABCDEF-token", "-100123456", "hello")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate limit")
}
