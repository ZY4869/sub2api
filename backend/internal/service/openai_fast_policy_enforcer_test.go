package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type fastPolicySettingRepoStub struct {
	values map[string]string
}

func (s *fastPolicySettingRepoStub) Get(_ context.Context, _ string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (s *fastPolicySettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	v, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}
func (s *fastPolicySettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}
func (s *fastPolicySettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			result[key] = v
		}
	}
	return result, nil
}
func (s *fastPolicySettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for k, v := range settings {
		s.values[k] = v
	}
	return nil
}
func (s *fastPolicySettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	if s.values == nil {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(s.values))
	for k, v := range s.values {
		result[k] = v
	}
	return result, nil
}
func (s *fastPolicySettingRepoStub) Delete(_ context.Context, key string) error {
	if s.values == nil {
		return nil
	}
	delete(s.values, key)
	return nil
}

func TestApplyOpenAIFastPolicyToJSONBody_FiltersAndPreservesUnsupportedFields(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{} // nil settingService -> default policy
	body := []byte(`{"model":"gpt-4.1","service_tier":"priority","unsupported_field":{"a":1}}`)

	next, decision, err := svc.applyOpenAIFastPolicyToJSONBody(context.Background(), nil, body, "priority", "gpt-4.1")
	require.NoError(t, err)
	require.True(t, decision.matched)
	require.Equal(t, OpenAIFastPolicyActionFilter, decision.action)

	require.False(t, gjson.GetBytes(next, "service_tier").Exists())
	require.Equal(t, int64(1), gjson.GetBytes(next, "unsupported_field.a").Int())
	require.Equal(t, "gpt-4.1", gjson.GetBytes(next, "model").String())
}

func TestApplyOpenAIFastPolicyToJSONBody_PassFlexByDefault(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{} // nil settingService -> default policy
	body := []byte(`{"model":"gpt-4.1","service_tier":"flex","unsupported_field":123}`)

	next, decision, err := svc.applyOpenAIFastPolicyToJSONBody(context.Background(), nil, body, "flex", "gpt-4.1")
	require.NoError(t, err)
	require.True(t, decision.matched)
	require.Equal(t, OpenAIFastPolicyActionPass, decision.action)
	require.JSONEq(t, string(body), string(next))
}

func TestApplyOpenAIFastPolicyToJSONBody_BlockWithCustomSettings(t *testing.T) {
	t.Parallel()

	repo := &fastPolicySettingRepoStub{
		values: map[string]string{
			SettingKeyOpenAIFastPolicySettings: `{"rules":[{"service_tier":"priority","action":"block","scope":"all"}]}`,
		},
	}
	settingSvc := &SettingService{settingRepo: repo}
	svc := &OpenAIGatewayService{settingService: settingSvc}

	body := []byte(`{"model":"gpt-4.1","service_tier":"priority"}`)
	_, decision, err := svc.applyOpenAIFastPolicyToJSONBody(context.Background(), nil, body, "priority", "gpt-4.1")
	require.Error(t, err)
	require.True(t, decision.matched)
	require.Equal(t, OpenAIFastPolicyActionBlock, decision.action)

	var blocked *openAIFastPolicyBlockedError
	require.ErrorAs(t, err, &blocked)
	require.Equal(t, "priority", blocked.ServiceTier)
	require.Equal(t, "gpt-4.1", blocked.Model)
}

func TestApplyOpenAIFastPolicyToRequestBodyMap_Filters(t *testing.T) {
	t.Parallel()

	svc := &OpenAIGatewayService{} // default policy
	reqBody := map[string]any{
		"model":        "gpt-4.1",
		"service_tier": "priority",
		"unsupported":  map[string]any{"a": 1},
	}

	modified, err := svc.applyOpenAIFastPolicyToRequestBodyMap(context.Background(), nil, reqBody)
	require.NoError(t, err)
	require.True(t, modified)
	_, ok := reqBody["service_tier"]
	require.False(t, ok)
	unsupported, ok := reqBody["unsupported"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 1, unsupported["a"])
}
