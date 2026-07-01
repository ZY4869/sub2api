package service

import (
	"regexp"
	"strings"
)

const publicModelCatalogExampleTitle = "模型库协议示例"

var (
	publicModelCatalogExampleFencePattern  = regexp.MustCompile(`^\s*(` + "```+" + `|~~~+)`)
	publicModelCatalogExampleTitlePattern  = regexp.MustCompile(`^#\s+(.+)$`)
	publicModelCatalogExamplePageIDPattern = regexp.MustCompile(`^##\s+(.+)$`)
)

var publicModelCatalogExamplePageOrder = []string{
	"common",
	"openai-native",
	"openai",
	"anthropic",
	"gemini",
	"grok",
	"antigravity",
	"vertex-batch",
}

var publicModelCatalogExamplePages = map[string]string{
	"common": `### OpenAI Responses
#### REST
` + "```bash" + `
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "请用一句话确认网关已经联通。"
  }'
` + "```" + `
`,
	"openai-native": `### Responses
#### JavaScript
` + "```javascript" + `
const response = await fetch("https://api.zyxai.de/v1/responses", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-5.4",
    input: "用一句话介绍这个模型。",
  }),
});

console.log(await response.json());
` + "```" + `
`,
	"openai": `### chat/completions
#### REST
` + "```bash" + `
curl https://api.zyxai.de/v1/chat/completions \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      { "role": "user", "content": "解释什么时候还应该使用 chat/completions。" }
    ]
  }'
` + "```" + `
`,
	"anthropic": `### messages
#### REST
` + "```bash" + `
curl https://api.zyxai.de/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "用一句话介绍这个模型。" }
    ]
  }'
` + "```" + `
`,
	"gemini": `### generateContent
#### REST
` + "```bash" + `
curl https://api.zyxai.de/v1beta/models/gemini-2.5-pro:generateContent \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      { "role": "user", "parts": [{ "text": "用一句话介绍这个模型。" }] }
    ]
  }'
` + "```" + `
`,
	"grok": `### Grok responses
#### REST
` + "```bash" + `
curl https://api.zyxai.de/grok/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4",
    "input": "用一句话介绍这个模型。"
  }'
` + "```" + `

### Grok messages
#### REST
` + "```bash" + `
curl https://api.zyxai.de/grok/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "grok-4",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "用一句话介绍这个模型。" }
    ]
  }'
` + "```" + `

### Grok count tokens
#### REST
` + "```bash" + `
curl https://api.zyxai.de/grok/v1/messages/count_tokens \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "grok-4",
    "messages": [
      { "role": "user", "content": "估算这句话的输入 tokens。" }
    ]
  }'
` + "```" + `
`,
	"antigravity": `### Antigravity messages
#### REST
` + "```bash" + `
curl https://api.zyxai.de/antigravity/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "antigravity-model",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "用一句话介绍这个模型。" }
    ]
  }'
` + "```" + `
`,
	"vertex-batch": `### Vertex / Batch jobs
#### REST
` + "```bash" + `
curl https://api.zyxai.de/vertex-batch/jobs \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-pro",
    "requests": [
      { "contents": [{ "role": "user", "parts": [{ "text": "用一句话介绍这个模型。" }] }] }
    ]
  }'
` + "```" + `
`,
}

type publicModelCatalogExampleDocument struct {
	Title string
	Pages map[string]string
}

func publicModelCatalogExampleTemplateMarkdown(pageID string, keywords []string) string {
	pageID = normalizePublicModelCatalogExamplePageID(pageID)
	if pageID == "" {
		return ""
	}
	return extractPublicModelCatalogExampleMarkdown(buildPublicModelCatalogExampleDocument(), pageID, keywords)
}

func buildPublicModelCatalogExampleDocument() string {
	return buildPublicModelCatalogExampleDocumentFromPages(publicModelCatalogExampleTitle, publicModelCatalogExamplePages)
}

func buildPublicModelCatalogExampleDocumentFromPages(title string, pages map[string]string) string {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		normalizedTitle = publicModelCatalogExampleTitle
	}
	lines := []string{"# " + normalizedTitle, ""}
	for index, pageID := range publicModelCatalogExamplePageOrder {
		lines = append(lines, "## "+pageID)
		body := strings.Trim(strings.ReplaceAll(pages[pageID], "\r\n", "\n"), "\n")
		if body != "" {
			lines = append(lines, strings.Split(body, "\n")...)
		}
		if index < len(publicModelCatalogExamplePageOrder)-1 {
			lines = append(lines, "")
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func buildPublicModelCatalogExamplePageSection(title string, pageID string, pageBody string) string {
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		normalizedTitle = publicModelCatalogExampleTitle
	}
	lines := []string{"# " + normalizedTitle, "", "## " + pageID}
	body := strings.Trim(strings.ReplaceAll(pageBody, "\r\n", "\n"), "\n")
	if body != "" {
		lines = append(lines, strings.Split(body, "\n")...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func parsePublicModelCatalogExampleDocument(content string) publicModelCatalogExampleDocument {
	normalized := normalizePublicModelCatalogExampleContent(content)
	if normalized == "" {
		return publicModelCatalogExampleDocument{Title: publicModelCatalogExampleTitle, Pages: map[string]string{}}
	}

	lines := strings.Split(normalized, "\n")
	title := publicModelCatalogExampleTitle
	pages := make(map[string][]string)
	currentPageID := ""
	inFence := false
	fenceMarker := ""

	for _, line := range lines {
		if title == publicModelCatalogExampleTitle {
			if match := publicModelCatalogExampleTitlePattern.FindStringSubmatch(line); match != nil {
				if candidate := strings.TrimSpace(match[1]); candidate != "" {
					title = candidate
				}
			}
		}
		if fence := parsePublicModelCatalogExampleFence(line); fence != "" {
			if !inFence {
				inFence = true
				fenceMarker = fence
			} else if matchesPublicModelCatalogExampleFence(line, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
		}
		if !inFence {
			if match := publicModelCatalogExamplePageIDPattern.FindStringSubmatch(line); match != nil {
				if pageID := normalizePublicModelCatalogExamplePageID(match[1]); pageID != "" {
					currentPageID = pageID
					if _, ok := pages[currentPageID]; !ok {
						pages[currentPageID] = []string{}
					}
					continue
				}
			}
		}
		if currentPageID != "" {
			pages[currentPageID] = append(pages[currentPageID], line)
		}
	}

	result := make(map[string]string, len(pages))
	for pageID, sectionLines := range pages {
		result[pageID] = strings.Trim(strings.Join(sectionLines, "\n"), "\n")
	}
	return publicModelCatalogExampleDocument{Title: title, Pages: result}
}

func normalizePublicModelCatalogExamplePageID(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	for _, pageID := range publicModelCatalogExamplePageOrder {
		if normalized == pageID {
			return normalized
		}
	}
	return ""
}

func parsePublicModelCatalogExampleFence(line string) string {
	match := publicModelCatalogExampleFencePattern.FindStringSubmatch(line)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func matchesPublicModelCatalogExampleFence(line string, fence string) bool {
	return strings.TrimSpace(line) == fence
}

func normalizePublicModelCatalogExampleContent(content string) string {
	return strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
}
