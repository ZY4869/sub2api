package service

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestBuildDefaultAPIDocsTemplateFromFS(t *testing.T) {
	docsFS := fstest.MapFS{
		"docs/index.md":               {Data: []byte("# API 文档中心\n")},
		"docs/pages/common.md":        {Data: []byte("## common\nCommon body\n")},
		"docs/pages/openai-native.md": {Data: []byte("## openai-native\nOpenAI Native body\n")},
		"docs/pages/openai.md":        {Data: []byte("## openai\nOpenAI body\n")},
		"docs/pages/anthropic.md":     {Data: []byte("## anthropic\nAnthropic body\n")},
		"docs/pages/gemini.md":        {Data: []byte("## gemini\nGemini body\n")},
		"docs/pages/grok.md":          {Data: []byte("## grok\nGrok body\n")},
		"docs/pages/antigravity.md":   {Data: []byte("## antigravity\nAntigravity body\n")},
		"docs/pages/vertex-batch.md":  {Data: []byte("## vertex-batch\nVertex body\n")},
		"docs/pages/document-ai.md":   {Data: []byte("## document-ai\nDocument AI body\n")},
	}

	content, err := buildDefaultAPIDocsTemplateFromFS(docsFS)
	require.NoError(t, err)

	expected := buildAPIDocsDocument("API 文档中心", map[string]string{
		"common":        "Common body",
		"openai-native": "OpenAI Native body",
		"openai":        "OpenAI body",
		"anthropic":     "Anthropic body",
		"gemini":        "Gemini body",
		"grok":          "Grok body",
		"antigravity":   "Antigravity body",
		"vertex-batch":  "Vertex body",
		"document-ai":   "Document AI body",
	})
	require.Equal(t, expected, content)
}

func TestBuildDefaultAPIDocsTemplateFromFS_RequiresAllPages(t *testing.T) {
	docsFS := fstest.MapFS{
		"docs/index.md":              {Data: []byte("# API 文档中心\n")},
		"docs/pages/common.md":       {Data: []byte("## common\nCommon body\n")},
		"docs/pages/openai.md":       {Data: []byte("## openai\nOpenAI body\n")},
		"docs/pages/anthropic.md":    {Data: []byte("## anthropic\nAnthropic body\n")},
		"docs/pages/gemini.md":       {Data: []byte("## gemini\nGemini body\n")},
		"docs/pages/grok.md":         {Data: []byte("## grok\nGrok body\n")},
		"docs/pages/vertex-batch.md": {Data: []byte("## vertex-batch\nVertex body\n")},
	}

	_, err := buildDefaultAPIDocsTemplateFromFS(docsFS)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing api docs page file for openai-native")
}

func TestBuildDefaultAPIDocsTemplateFromFS_RejectsMismatchedPageHeader(t *testing.T) {
	docsFS := fstest.MapFS{
		"docs/index.md":               {Data: []byte("# API 文档中心\n")},
		"docs/pages/common.md":        {Data: []byte("## common\nCommon body\n")},
		"docs/pages/openai-native.md": {Data: []byte("## openai-native\nOpenAI Native body\n")},
		"docs/pages/openai.md":        {Data: []byte("## openai\nOpenAI body\n")},
		"docs/pages/anthropic.md":     {Data: []byte("## anthropic\nAnthropic body\n")},
		"docs/pages/document-ai.md":   {Data: []byte("## document-ai\nDocument AI body\n")},
		"docs/pages/gemini.md":        {Data: []byte("## openai\nWrong header\n")},
		"docs/pages/grok.md":          {Data: []byte("## grok\nGrok body\n")},
		"docs/pages/antigravity.md":   {Data: []byte("## antigravity\nAntigravity body\n")},
		"docs/pages/vertex-batch.md":  {Data: []byte("## vertex-batch\nVertex body\n")},
	}

	_, err := buildDefaultAPIDocsTemplateFromFS(docsFS)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api docs page header mismatch")
}
