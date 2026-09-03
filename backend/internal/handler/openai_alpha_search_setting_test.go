package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type alphaSearchHandlerSettingRepo struct {
	value string
	err   error
}

func (r *alphaSearchHandlerSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (r *alphaSearchHandlerSettingRepo) GetValue(context.Context, string) (string, error) {
	return r.value, r.err
}

func (r *alphaSearchHandlerSettingRepo) Set(context.Context, string, string) error { return nil }

func (r *alphaSearchHandlerSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

func (r *alphaSearchHandlerSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (r *alphaSearchHandlerSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}

func (r *alphaSearchHandlerSettingRepo) Delete(context.Context, string) error { return nil }

func TestOpenAIGatewayHandlerAlphaSearchGlobalSwitch(t *testing.T) {
	tests := []struct {
		name        string
		repo        *alphaSearchHandlerSettingRepo
		wantStatus  int
		wantType    string
		wantMessage string
	}{
		{name: "disabled returns not found", repo: &alphaSearchHandlerSettingRepo{value: "false"}, wantStatus: http.StatusNotFound, wantType: "not_found_error", wantMessage: "OpenAI alpha/search is disabled"},
		{name: "repository failure returns unavailable", repo: &alphaSearchHandlerSettingRepo{err: errors.New("database unavailable")}, wantStatus: http.StatusServiceUnavailable, wantType: "api_error", wantMessage: "OpenAI alpha/search availability could not be determined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newOpenAIEmbeddingsHandlerTestContext(`{"model":"gpt-5.4","commands":{"search_query":[{"q":"news"}]}}`)
			attachOpenAIEmbeddingsAuth(c, service.PlatformOpenAI)
			h := &OpenAIGatewayHandler{settingService: service.NewSettingService(tt.repo, &config.Config{})}

			h.AlphaSearch(c)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.JSONEq(t, `{"error":{"type":"`+tt.wantType+`","message":"`+tt.wantMessage+`"}}`, rec.Body.String())
		})
	}
}
