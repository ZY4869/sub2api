package service

import (
	"encoding/json"
	"fmt"
)

func applyCodexOAuthTransformToJSON(body []byte, isCodexCLI bool, isCompact bool) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("unmarshal for codex transform: %w", err)
	}
	result := applyCodexOAuthTransform(reqBody, isCodexCLI, isCompact)
	if !result.Modified {
		return body, nil
	}
	updated, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("remarshal after codex transform: %w", err)
	}
	return updated, nil
}
