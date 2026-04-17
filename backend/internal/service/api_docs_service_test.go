package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type apiDocsRepoStub struct {
	values map[string]string
}

func (s *apiDocsRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *apiDocsRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *apiDocsRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *apiDocsRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *apiDocsRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *apiDocsRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *apiDocsRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestAPIDocsService_FallsBackToDefaultTemplate(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	document, err := service.GetDocument(context.Background())
	require.NoError(t, err)
	require.False(t, document.HasOverride)
	require.Equal(t, document.DefaultContent, document.EffectiveContent)
	require.Contains(t, document.EffectiveContent, "# API 文档中心")
}

func TestAPIDocsService_SaveOverrideAndClear(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	err := service.SaveOverride(context.Background(), "  # 自定义文档\r\n\r\n内容  ")
	require.NoError(t, err)
	require.Equal(t, "# 自定义文档\n\n内容\n", repo.values[SettingKeyAPIDocsMarkdown])

	document, err := service.GetDocument(context.Background())
	require.NoError(t, err)
	require.True(t, document.HasOverride)
	require.Equal(t, "# 自定义文档\n\n内容\n", document.EffectiveContent)

	err = service.ClearOverride(context.Background())
	require.NoError(t, err)

	document, err = service.GetDocument(context.Background())
	require.NoError(t, err)
	require.False(t, document.HasOverride)
	require.Equal(t, document.DefaultContent, document.EffectiveContent)
}

func TestAPIDocsService_RejectsBlankOverride(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	err := service.SaveOverride(context.Background(), "   \r\n\t ")
	require.ErrorIs(t, err, ErrAPIDocsEmptyContent)
}

func TestAPIDocsService_ConstructorIsWireCompatible(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	require.NotNil(t, service)
	require.NotNil(t, NewSettingService(repo, &config.Config{}))
}
