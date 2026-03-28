package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"

type TLSFingerprintProfile = tlsfingerprint.Profile

func defaultTLSFingerprintProfile() *tlsfingerprint.Profile {
	if profile := tlsfingerprint.GlobalRegistry().GetDefaultProfile(); profile != nil {
		return profile
	}
	return &tlsfingerprint.Profile{Name: "Built-in Default (Node.js 24.x)"}
}

func resolveAccountTLSFingerprintProfile(account *Account, service *TLSFingerprintProfileService) *tlsfingerprint.Profile {
	if account == nil || !account.IsTLSFingerprintEnabled() {
		return nil
	}
	if service != nil {
		return service.ResolveTLSProfile(account)
	}
	return defaultTLSFingerprintProfile()
}
