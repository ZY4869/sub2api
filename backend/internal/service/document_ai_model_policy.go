package service

import (
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func documentAIModelForbiddenError() error {
	return infraerrors.Forbidden(
		"document_ai_model_forbidden",
		"This model is not allowed for this Baidu Document AI group. Please contact the administrator to update account model restrictions.",
	)
}

func documentAIUnsupportedModelError(mode string) error {
	if mode == DocumentAIJobModeDirect {
		return infraerrors.BadRequest("document_ai_invalid_request", "unsupported direct document ai model")
	}
	return infraerrors.BadRequest("document_ai_invalid_request", "unsupported document ai model")
}

func documentAIModelSupportsMode(mode string, model string) bool {
	switch mode {
	case DocumentAIJobModeDirect:
		return DocumentAIModelSupportsDirect(model)
	case DocumentAIJobModeAsync:
		return DocumentAIModelSupportsAsync(model)
	default:
		return false
	}
}

func documentAIResolveAccountDisplayMapping(account *Account) (mapping map[string]string, restricted bool) {
	if account == nil {
		return nil, false
	}
	scope, ok := ExtractAccountModelScopeV2(account.Extra)
	if !ok || scope == nil || len(scope.Entries) == 0 {
		return nil, false
	}

	mapping = make(map[string]string, len(scope.Entries))
	for _, entry := range scope.Entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		normalizedTarget := normalizeDocumentAIModelID(targetModelID)
		if normalizedTarget == "" {
			// Only keep routable Document AI models in this projection.
			continue
		}
		mapping[displayModelID] = normalizedTarget
	}
	return mapping, true
}

func documentAIUnionModelsForAccounts(accounts []Account) []DocumentAIModelDescriptor {
	base := BuiltinDocumentAIModels()
	baseByID := make(map[string]DocumentAIModelDescriptor, len(base))
	for _, item := range base {
		baseByID[item.ID] = item
	}

	merged := make(map[string]DocumentAIModelDescriptor)
	merge := func(displayID string, baseDescriptor DocumentAIModelDescriptor) {
		displayID = strings.TrimSpace(displayID)
		if displayID == "" {
			return
		}
		next := baseDescriptor
		next.ID = displayID

		if current, ok := merged[displayID]; ok {
			current.Modes = unionStringSlice(current.Modes, next.Modes)
			current.SupportsMultipart = current.SupportsMultipart || next.SupportsMultipart
			current.SupportsFileURL = current.SupportsFileURL || next.SupportsFileURL
			current.SupportsFileBase64 = current.SupportsFileBase64 || next.SupportsFileBase64
			merged[displayID] = current
			return
		}
		merged[displayID] = next
	}

	for i := range accounts {
		account := &accounts[i]
		mapping, restricted := documentAIResolveAccountDisplayMapping(account)
		if !restricted {
			for _, item := range base {
				merge(item.ID, item)
			}
			continue
		}
		for displayID, targetID := range mapping {
			if baseDescriptor, ok := baseByID[targetID]; ok {
				merge(displayID, baseDescriptor)
			}
		}
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]DocumentAIModelDescriptor, 0, len(keys))
	for _, key := range keys {
		out = append(out, merged[key])
	}
	return out
}

func unionStringSlice(left []string, right []string) []string {
	if len(left) == 0 {
		return append([]string(nil), right...)
	}
	seen := make(map[string]struct{}, len(left)+len(right))
	out := make([]string, 0, len(left)+len(right))
	for _, item := range left {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for _, item := range right {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func documentAISelectAccountForDisplayModel(accounts []Account, rawDisplayModelID string, mode string) (*Account, string, error) {
	displayModelID := strings.TrimSpace(rawDisplayModelID)
	if displayModelID == "" {
		return nil, "", documentAIUnsupportedModelError(mode)
	}
	// Backwards compatibility for built-in model aliases.
	if canonical := normalizeDocumentAIModelID(displayModelID); canonical != "" {
		displayModelID = canonical
	}

	supported := normalizeDocumentAIModelID(displayModelID) != ""
	allowed := false
	allowedInMode := false

	for i := range accounts {
		account := &accounts[i]
		if account == nil || !account.IsBaiduDocumentAI() {
			continue
		}

		mapping, restricted := documentAIResolveAccountDisplayMapping(account)
		if !restricted {
			target := normalizeDocumentAIModelID(displayModelID)
			if target == "" {
				continue
			}
			supported = true
			allowed = true
			if !documentAIModelSupportsMode(mode, target) {
				continue
			}
			allowedInMode = true

			if mode == DocumentAIJobModeDirect {
				if documentAIAccountNeedDirect(target)(account) {
					return account, target, nil
				}
				continue
			}
			if documentAIAccountNeedAsync(account) {
				return account, target, nil
			}
			continue
		}

		target, ok := mapping[displayModelID]
		if !ok {
			continue
		}
		supported = true
		allowed = true
		if !documentAIModelSupportsMode(mode, target) {
			continue
		}
		allowedInMode = true

		if mode == DocumentAIJobModeDirect {
			if documentAIAccountNeedDirect(target)(account) {
				return account, target, nil
			}
			continue
		}
		if documentAIAccountNeedAsync(account) {
			return account, target, nil
		}
	}

	if !supported {
		return nil, "", documentAIUnsupportedModelError(mode)
	}
	if !allowed {
		return nil, "", documentAIModelForbiddenError()
	}
	if !allowedInMode {
		return nil, "", documentAIUnsupportedModelError(mode)
	}
	return nil, "", infraerrors.ServiceUnavailable("document_ai_provider_error", "no available baidu document ai account")
}
