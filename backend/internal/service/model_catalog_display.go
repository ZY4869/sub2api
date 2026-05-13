package service

import (
	"regexp"
	"strings"
)

var (
	modelCatalogDateVersionSuffixPattern = regexp.MustCompile(`-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$`)
	openAIReasoningModelPattern          = regexp.MustCompile(`^o\d`)
)

var modelCatalogTokenOverrides = map[string]string{
	"abab":        "ABAB",
	"airx":        "AirX",
	"aya":         "Aya",
	"c4ai":        "C4AI",
	"chatgpt":     "ChatGPT",
	"chatglm":     "ChatGLM",
	"cogvideo":    "CogVideo",
	"cogview":     "CogView",
	"codestral":   "Codestral",
	"codellama":   "CodeLlama",
	"deepseek":    "DeepSeek",
	"distill":     "Distill",
	"doubao":      "Doubao",
	"ernie":       "ERNIE",
	"flash":       "Flash",
	"glm":         "GLM",
	"hunyuan":     "Hunyuan",
	"kimi":        "Kimi",
	"latest":      "Latest",
	"lite":        "Lite",
	"llama":       "Llama",
	"longcontext": "LongContext",
	"max":         "Max",
	"medium":      "Medium",
	"mistral":     "Mistral",
	"mini":        "Mini",
	"mixtral":     "Mixtral",
	"moonshot":    "Moonshot",
	"nano":        "Nano",
	"online":      "Online",
	"open":        "Open",
	"oss":         "OSS",
	"pixtral":     "Pixtral",
	"plus":        "Plus",
	"preview":     "Preview",
	"pro":         "Pro",
	"qwen":        "Qwen",
	"qwq":         "QwQ",
	"r1":          "R1",
	"rag":         "RAG",
	"reasoner":    "Reasoner",
	"realtime":    "Realtime",
	"small":       "Small",
	"sonar":       "Sonar",
	"spark":       "Spark",
	"speed":       "Speed",
	"std":         "STD",
	"tab":         "Tab",
	"thinking":    "Thinking",
	"tiny":        "Tiny",
	"tools":       "Tools",
	"turbo":       "Turbo",
	"ultra":       "Ultra",
	"vision":      "Vision",
	"yi":          "Yi",
}

func NormalizeModelCatalogModelID(model string) string {
	return normalizeModelCatalogAlias(model)
}

func FormatModelCatalogDisplayName(model string) string {
	canonical := NormalizeModelCatalogModelID(model)
	if canonical == "" {
		return ""
	}
	parts := strings.FieldsFunc(canonical, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return canonical
	}
	formatted := make([]string, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		current := parts[index]
		if shouldMergeModelCatalogVersionToken(current, parts, index) {
			formatted = append(formatted, current+"."+parts[index+1])
			index++
			continue
		}
		formatted = append(formatted, formatModelCatalogToken(current, len(formatted) == 0))
	}
	return strings.Join(formatted, " ")
}

func InferModelCatalogIconKey(model string) string {
	canonical := NormalizeModelCatalogModelID(model)
	switch {
	case strings.HasPrefix(canonical, "claude"):
		return "claude"
	case strings.HasPrefix(canonical, "gemini"):
		return "gemini"
	case strings.HasPrefix(canonical, "gpt"), strings.HasPrefix(canonical, "codex"), openAIReasoningModelPattern.MatchString(canonical):
		return "chatgpt"
	default:
		return ""
	}
}

func formatModelCatalogBrand(value string) string {
	if override, ok := modelCatalogTokenOverrides[value]; ok {
		return override
	}
	switch value {
	case "claude":
		return "Claude"
	case "gpt":
		return "GPT"
	case "gemini":
		return "Gemini"
	case "codex":
		return "Codex"
	default:
		if openAIReasoningModelPattern.MatchString(value) {
			return strings.ToUpper(value)
		}
		if value == "" {
			return value
		}
		return strings.ToUpper(value[:1]) + value[1:]
	}
}

func formatModelCatalogToken(value string, isFirst bool) string {
	if isFirst {
		return formatModelCatalogBrand(value)
	}
	if override, ok := modelCatalogTokenOverrides[value]; ok {
		return override
	}
	if value == "" {
		return value
	}
	if value[0] < 'a' || value[0] > 'z' {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func shouldMergeModelCatalogVersionToken(current string, parts []string, index int) bool {
	if index+1 >= len(parts) {
		return false
	}
	return isShortModelCatalogNumericToken(current) && isShortModelCatalogNumericToken(parts[index+1])
}

func isShortModelCatalogNumericToken(value string) bool {
	if len(value) == 0 || len(value) > 2 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
