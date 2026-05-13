package service

import (
	"fmt"
	"regexp"
	"strings"
)

var antigravityUserAgentVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func NormalizeAntigravityUserAgentVersion(raw string) (string, error) {
	version := strings.TrimSpace(raw)
	if version == "" {
		return "", nil
	}
	if !antigravityUserAgentVersionPattern.MatchString(version) {
		return "", fmt.Errorf("antigravity user-agent version must be empty or a valid semver like 1.21.9")
	}
	return version, nil
}
