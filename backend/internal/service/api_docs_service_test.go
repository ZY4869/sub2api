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
	require.Contains(t, document.EffectiveContent, "## common")
	require.Contains(t, document.EffectiveContent, "## gemini")
}

func TestAPIDocsService_SaveOverrideAndClear(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	override := "# Custom Docs\r\n\r\n## common\r\n### Intro\r\nHello"
	err := service.SaveOverride(context.Background(), override)
	require.NoError(t, err)
	require.Equal(t, "# Custom Docs\n\n## common\n### Intro\nHello\n", repo.values[SettingKeyAPIDocsMarkdown])

	document, err := service.GetDocument(context.Background())
	require.NoError(t, err)
	require.True(t, document.HasOverride)
	require.Equal(t, "# Custom Docs\n\n## common\n### Intro\nHello\n", document.EffectiveContent)

	err = service.ClearOverride(context.Background())
	require.NoError(t, err)
	require.Empty(t, repo.values)

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

func TestAPIDocsService_GetPageDocumentFallsBackToDefaultSection(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	document, err := service.GetPageDocument(context.Background(), "gemini")
	require.NoError(t, err)
	require.False(t, document.HasOverride)
	require.Equal(t, document.DefaultContent, document.EffectiveContent)
	require.Contains(t, document.EffectiveContent, "## gemini")
	require.NotContains(t, document.EffectiveContent, "## openai")
}

func TestAPIDocsService_SupportsOpenAINativePageOverride(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	override := "# API Docs\n\n## openai-native\n### Responses\nOpenAI native only"
	err := service.SavePageOverride(context.Background(), "openai-native", override)
	require.NoError(t, err)

	document, err := service.GetPageDocument(context.Background(), "openai-native")
	require.NoError(t, err)
	require.True(t, document.HasOverride)
	require.Contains(t, document.EffectiveContent, "## openai-native")
	require.Contains(t, document.EffectiveContent, "### Responses")
	require.Contains(t, repo.values[apiDocsPageSettingKey("openai-native")], "## openai-native")
}

func TestAPIDocsService_SupportsDocumentAIPageOverride(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	override := "# API Docs\n\n## document-ai\n### Runtime\nDocument AI only"
	err := service.SavePageOverride(context.Background(), "document-ai", override)
	require.NoError(t, err)

	document, err := service.GetPageDocument(context.Background(), "document-ai")
	require.NoError(t, err)
	require.True(t, document.HasOverride)
	require.Contains(t, document.EffectiveContent, "## document-ai")
	require.Contains(t, document.EffectiveContent, "### Runtime")
	require.Contains(t, repo.values[apiDocsPageSettingKey("document-ai")], "## document-ai")
}

func TestAPIDocsService_SavePageOverrideOnlyChangesThatPage(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	override := "# API Docs\n\n## gemini\n### Custom Section\nGemini only"
	err := service.SavePageOverride(context.Background(), "gemini", override)
	require.NoError(t, err)

	geminiDoc, err := service.GetPageDocument(context.Background(), "gemini")
	require.NoError(t, err)
	require.True(t, geminiDoc.HasOverride)
	require.Contains(t, geminiDoc.EffectiveContent, "### Custom Section")

	openAIDoc, err := service.GetPageDocument(context.Background(), "openai")
	require.NoError(t, err)
	require.False(t, openAIDoc.HasOverride)
	require.NotContains(t, openAIDoc.EffectiveContent, "### Custom Section")

	fullDoc, err := service.GetDocument(context.Background())
	require.NoError(t, err)
	require.True(t, fullDoc.HasOverride)
	require.Contains(t, fullDoc.EffectiveContent, "## gemini\n### Custom Section\nGemini only")
	require.Contains(t, fullDoc.EffectiveContent, "## openai")
}

func TestAPIDocsService_GetPageDocumentFallsBackToLegacyBlob(t *testing.T) {
	repo := &apiDocsRepoStub{
		values: map[string]string{
			SettingKeyAPIDocsMarkdown: "# Legacy Docs\n\n## common\n### Common Intro\nLegacy common\n\n## gemini\n### Legacy Gemini\nLegacy section\n",
		},
	}
	service := NewAPIDocsService(repo)

	document, err := service.GetPageDocument(context.Background(), "gemini")
	require.NoError(t, err)
	require.True(t, document.HasOverride)
	require.Contains(t, document.EffectiveContent, "# Legacy Docs")
	require.Contains(t, document.EffectiveContent, "## gemini")
	require.Contains(t, document.EffectiveContent, "### Legacy Gemini")
}

func TestAPIDocsService_ClearPageOverrideWritesDefaultSectionAndShadowsLegacy(t *testing.T) {
	repo := &apiDocsRepoStub{
		values: map[string]string{
			SettingKeyAPIDocsMarkdown: "# Legacy Docs\n\n## common\n### Common Intro\nLegacy common\n\n## gemini\n### Legacy Gemini\nLegacy section\n",
		},
	}
	service := NewAPIDocsService(repo)

	err := service.ClearPageOverride(context.Background(), "gemini")
	require.NoError(t, err)

	defaultGemini, err := service.GetPageDocument(context.Background(), "gemini")
	require.NoError(t, err)
	require.False(t, defaultGemini.HasOverride)
	require.Equal(t, defaultGemini.DefaultContent, defaultGemini.EffectiveContent)
	require.Equal(t, defaultGemini.DefaultContent, repo.values[apiDocsPageSettingKey("gemini")])

	fullDoc, err := service.GetDocument(context.Background())
	require.NoError(t, err)
	require.True(t, fullDoc.HasOverride)
	require.Contains(t, fullDoc.EffectiveContent, "Legacy common")
	require.NotContains(t, fullDoc.EffectiveContent, "### Legacy Gemini")
}

func TestAPIDocsService_ClearOverrideRemovesLegacyAndPageKeys(t *testing.T) {
	repo := &apiDocsRepoStub{
		values: map[string]string{
			SettingKeyAPIDocsMarkdown:          "# Legacy Docs\n",
			apiDocsPageSettingKey("gemini"):    "# API Docs\n\n## gemini\n### Override\nGemini\n",
			apiDocsPageSettingKey("anthropic"): "# API Docs\n\n## anthropic\n### Override\nAnthropic\n",
		},
	}
	service := NewAPIDocsService(repo)

	err := service.ClearOverride(context.Background())
	require.NoError(t, err)
	require.Empty(t, repo.values)
}

func TestAPIDocsService_RejectsInvalidPageID(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	_, err := service.GetPageDocument(context.Background(), "invalid")
	require.ErrorIs(t, err, ErrAPIDocsInvalidPage)

	err = service.SavePageOverride(context.Background(), "invalid", "# API Docs")
	require.ErrorIs(t, err, ErrAPIDocsInvalidPage)

	err = service.ClearPageOverride(context.Background(), "invalid")
	require.ErrorIs(t, err, ErrAPIDocsInvalidPage)
}

func TestAPIDocsService_ConstructorIsWireCompatible(t *testing.T) {
	repo := &apiDocsRepoStub{values: map[string]string{}}
	service := NewAPIDocsService(repo)

	require.NotNil(t, service)
	require.NotNil(t, NewSettingService(repo, &config.Config{}))
}
