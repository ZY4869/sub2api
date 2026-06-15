package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorRepositoryCreatePersistsDualModeFields(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	mock.ExpectQuery("INSERT INTO channel_monitors").
		WithArgs(
			"Pool health",
			service.ChannelMonitorProviderOpenAI,
			"",
			service.ChannelMonitorProbeModeAccountPool,
			service.ChannelMonitorRequestProtocolOpenAI,
			"",
			60,
			true,
			sqlmock.AnyArg(),
			"shared-main",
			sqlmock.AnyArg(),
			jsonStringArg{expected: map[string]string{"shared-main": "anthropic", "shared-side": "gemini"}},
			service.ChannelMonitorModelProbeStrategyAllSelected,
			"只回复 {{challenge}}",
			nil,
			jsonStringArg{expected: map[string]string{"x-debug": "1"}},
			service.ChannelMonitorBodyOverrideModeMerge,
			jsonAnyMapArg{expected: map[string]any{"temperature": float64(0)}},
			service.ChannelMonitorOpenAIAPIModeChatCompletions,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(7), now, now))

	mock.ExpectQuery("SELECT").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"provider",
			"endpoint",
			"probe_mode",
			"request_protocol",
			"api_key_encrypted",
			"interval_seconds",
			"enabled",
			"account_ids",
			"primary_model_id",
			"additional_model_ids",
			"model_source_protocols",
			"model_probe_strategy",
			"test_prompt_template",
			"template_id",
			"extra_headers",
			"body_override_mode",
			"body_override",
			"openai_api_mode",
			"last_run_at",
			"next_run_at",
			"created_at",
			"updated_at",
		}).AddRow(
			int64(7),
			"Pool health",
			service.ChannelMonitorProviderOpenAI,
			"",
			service.ChannelMonitorProbeModeAccountPool,
			service.ChannelMonitorRequestProtocolOpenAI,
			nil,
			60,
			true,
			"{101,102}",
			"shared-main",
			"{shared-side}",
			[]byte(`{"shared-main":"anthropic","shared-side":"gemini"}`),
			service.ChannelMonitorModelProbeStrategyAllSelected,
			"只回复 {{challenge}}",
			nil,
			[]byte(`{"x-debug":"1"}`),
			service.ChannelMonitorBodyOverrideModeMerge,
			[]byte(`{"temperature":0}`),
			service.ChannelMonitorOpenAIAPIModeChatCompletions,
			now,
			now.Add(time.Minute),
			now,
			now,
		))

	repo := NewChannelMonitorRepository(db)
	lastRun := now
	nextRun := now.Add(time.Minute)
	created, err := repo.Create(context.Background(), &service.ChannelMonitor{
		Name:                 "Pool health",
		Provider:             service.ChannelMonitorProviderOpenAI,
		ProbeMode:            service.ChannelMonitorProbeModeAccountPool,
		RequestProtocol:      service.ChannelMonitorRequestProtocolOpenAI,
		IntervalSeconds:      60,
		Enabled:              true,
		AccountIDs:           []int64{101, 102},
		PrimaryModelID:       "shared-main",
		AdditionalModelIDs:   []string{"shared-side"},
		ModelSourceProtocols: map[string]string{"shared-main": "anthropic", "shared-side": "gemini"},
		ModelProbeStrategy:   service.ChannelMonitorModelProbeStrategyAllSelected,
		TestPromptTemplate:   "只回复 {{challenge}}",
		ExtraHeaders:         map[string]string{"x-debug": "1"},
		BodyOverrideMode:     service.ChannelMonitorBodyOverrideModeMerge,
		BodyOverride:         map[string]any{"temperature": 0},
		OpenAIAPIMode:        service.ChannelMonitorOpenAIAPIModeChatCompletions,
		LastRunAt:            &lastRun,
		NextRunAt:            &nextRun,
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), created.ID)
	require.Equal(t, map[string]string{"shared-main": "anthropic", "shared-side": "gemini"}, created.ModelSourceProtocols)
	require.NoError(t, mock.ExpectationsWereMet())
}

type jsonStringArg struct {
	expected map[string]string
}

func (a jsonStringArg) Match(v driver.Value) bool {
	raw, ok := v.(string)
	if !ok {
		return false
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false
	}
	return reflect.DeepEqual(a.expected, parsed)
}

type jsonAnyMapArg struct {
	expected map[string]any
}

func (a jsonAnyMapArg) Match(v driver.Value) bool {
	raw, ok := v.(string)
	if !ok {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false
	}
	return reflect.DeepEqual(a.expected, parsed)
}
