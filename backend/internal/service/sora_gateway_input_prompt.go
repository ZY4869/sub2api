package service

import (
	"fmt"
	"regexp"
	"strings"
)

var soraStoryboardPattern = regexp.MustCompile(`\[\d+(?:\.\d+)?s\]`)
var soraStoryboardShotPattern = regexp.MustCompile(`\[(\d+(?:\.\d+)?)s\]\s*([^\[]+)`)
var soraRemixTargetPattern = regexp.MustCompile(`s_[a-f0-9]{32}`)
var soraRemixTargetInURLPattern = regexp.MustCompile(`https://sora\.chatgpt\.com/p/s_[a-f0-9]{32}`)

func extractSoraInput(body map[string]any) (prompt, imageInput, videoInput, remixTargetID string) {
	if body == nil {
		return "", "", "", ""
	}
	if v, ok := body["remix_target_id"].(string); ok {
		remixTargetID = strings.TrimSpace(v)
	}
	if v, ok := body["image"].(string); ok {
		imageInput = v
	}
	if v, ok := body["video"].(string); ok {
		videoInput = v
	}
	if v, ok := body["prompt"].(string); ok && strings.TrimSpace(v) != "" {
		prompt = v
	}
	if messages, ok := body["messages"].([]any); ok {
		builder := strings.Builder{}
		for _, raw := range messages {
			msg, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role != "" && role != "user" {
				continue
			}
			content := msg["content"]
			text, img, vid := parseSoraMessageContent(content)
			if text != "" {
				if builder.Len() > 0 {
					_, _ = builder.WriteString("\n")
				}
				_, _ = builder.WriteString(text)
			}
			if imageInput == "" && img != "" {
				imageInput = img
			}
			if videoInput == "" && vid != "" {
				videoInput = vid
			}
		}
		if prompt == "" {
			prompt = builder.String()
		}
	}
	if remixTargetID == "" {
		remixTargetID = extractRemixTargetIDFromPrompt(prompt)
	}
	prompt = cleanRemixLinkFromPrompt(prompt)
	return prompt, imageInput, videoInput, remixTargetID
}

func parseSoraMessageContent(content any) (text, imageInput, videoInput string) {
	switch val := content.(type) {
	case string:
		return val, "", ""
	case []any:
		builder := strings.Builder{}
		for _, item := range val {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			t, _ := itemMap["type"].(string)
			switch t {
			case "text":
				if txt, ok := itemMap["text"].(string); ok && strings.TrimSpace(txt) != "" {
					if builder.Len() > 0 {
						_, _ = builder.WriteString("\n")
					}
					_, _ = builder.WriteString(txt)
				}
			case "image_url":
				if imageInput == "" {
					if urlVal, ok := itemMap["image_url"].(map[string]any); ok {
						imageInput = fmt.Sprintf("%v", urlVal["url"])
					} else if urlStr, ok := itemMap["image_url"].(string); ok {
						imageInput = urlStr
					}
				}
			case "video_url":
				if videoInput == "" {
					if urlVal, ok := itemMap["video_url"].(map[string]any); ok {
						videoInput = fmt.Sprintf("%v", urlVal["url"])
					} else if urlStr, ok := itemMap["video_url"].(string); ok {
						videoInput = urlStr
					}
				}
			}
		}
		return builder.String(), imageInput, videoInput
	default:
		return "", "", ""
	}
}

func isSoraStoryboardPrompt(prompt string) bool {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return false
	}
	return len(soraStoryboardPattern.FindAllString(prompt, -1)) >= 1
}

func formatSoraStoryboardPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	matches := soraStoryboardShotPattern.FindAllStringSubmatch(prompt, -1)
	if len(matches) == 0 {
		return prompt
	}
	firstBracketPos := strings.Index(prompt, "[")
	instructions := ""
	if firstBracketPos > 0 {
		instructions = strings.TrimSpace(prompt[:firstBracketPos])
	}
	shots := make([]string, 0, len(matches))
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}
		duration := strings.TrimSpace(match[1])
		scene := strings.TrimSpace(match[2])
		if scene == "" {
			continue
		}
		shots = append(shots, fmt.Sprintf("Shot %d:\nduration: %ssec\nScene: %s", i+1, duration, scene))
	}
	if len(shots) == 0 {
		return prompt
	}
	timeline := strings.Join(shots, "\n\n")
	if instructions == "" {
		return timeline
	}
	return fmt.Sprintf("current timeline:\n%s\n\ninstructions:\n%s", timeline, instructions)
}

func extractRemixTargetIDFromPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	return strings.TrimSpace(soraRemixTargetPattern.FindString(prompt))
}

func cleanRemixLinkFromPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return prompt
	}
	cleaned := soraRemixTargetInURLPattern.ReplaceAllString(prompt, "")
	cleaned = soraRemixTargetPattern.ReplaceAllString(cleaned, "")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}
