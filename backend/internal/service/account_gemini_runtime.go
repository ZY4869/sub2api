package service

import "strings"

func (a *Account) IsGemini() bool {
	return EffectiveProtocol(a) == PlatformGemini
}

func (a *Account) GeminiOAuthType() string {
	if EffectiveProtocol(a) != PlatformGemini || a.Type != AccountTypeOAuth {
		return ""
	}
	oauthType := strings.TrimSpace(a.GetCredential("oauth_type"))
	if oauthType == "" {
		if strings.TrimSpace(a.GetCredential("vertex_project_id")) != "" {
			return "vertex_ai"
		}
		if strings.TrimSpace(a.GetCredential("project_id")) != "" {
			return "code_assist"
		}
	}
	return oauthType
}

func (a *Account) GeminiTierID() string {
	tierID := strings.TrimSpace(a.GetCredential("tier_id"))
	if canonical := canonicalGeminiTierID(tierID); canonical != "" {
		return canonical
	}
	return tierID
}

func (a *Account) IsGeminiCodeAssist() bool {
	if EffectiveProtocol(a) != PlatformGemini || a.Type != AccountTypeOAuth {
		return false
	}
	oauthType := a.GeminiOAuthType()
	if oauthType == "" {
		return strings.TrimSpace(a.GetCredential("project_id")) != ""
	}
	return oauthType == "code_assist"
}
