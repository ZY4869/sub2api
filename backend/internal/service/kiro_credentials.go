package service

import "strings"

func NormalizeKiroCredentialsForStorage(credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		return credentials
	}
	cloned := make(map[string]any, len(credentials))
	for key, value := range credentials {
		cloned[key] = value
	}
	NormalizeKiroCredentialsInPlace(cloned)
	return cloned
}

func NormalizeKiroCredentialsInPlace(credentials map[string]any) bool {
	if len(credentials) == 0 {
		return false
	}
	changed := false

	if region := strings.TrimSpace(kiroCredentialStringValue(credentials["api_region"])); region != "" {
		if credentials["api_region"] != region {
			credentials["api_region"] = region
			changed = true
		}
		if _, hasLegacy := credentials["region"]; hasLegacy {
			delete(credentials, "region")
			changed = true
		}
	} else if legacy := strings.TrimSpace(kiroCredentialStringValue(credentials["region"])); legacy != "" {
		credentials["api_region"] = legacy
		delete(credentials, "region")
		changed = true
	}

	for _, key := range []string{"profile_arn", "auth_method", "client_id", "client_secret", "client_id_hash", "start_url"} {
		if value := strings.TrimSpace(kiroCredentialStringValue(credentials[key])); value != "" && credentials[key] != value {
			credentials[key] = value
			changed = true
		}
	}

	return changed
}

func NormalizeKiroAccountCredentials(account *Account) bool {
	if account == nil || account.Platform != PlatformKiro {
		return false
	}
	if account.Credentials == nil {
		account.Credentials = map[string]any{}
	}
	return NormalizeKiroCredentialsInPlace(account.Credentials)
}

func KiroStoredRegion(account *Account) string {
	if account == nil {
		return ""
	}
	if region := strings.TrimSpace(account.GetCredential("api_region")); region != "" {
		return region
	}
	return strings.TrimSpace(account.GetCredential("region"))
}

func kiroCredentialStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case jsonNumberLike:
		return v.String()
	default:
		return ""
	}
}

type jsonNumberLike interface {
	String() string
}
