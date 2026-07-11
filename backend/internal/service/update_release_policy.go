package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"runtime"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const allowedDownloadHost = "github.com"

var releaseSemverPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

// validateDownloadURL checks if the URL is from the configured trusted release repository.
// SECURITY: This prevents SSRF and blocks upstream LGPL or unrelated release assets.
func validateDownloadURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}
	if !isTrustedReleaseAssetURL(parsedURL) {
		return fmt.Errorf("download from untrusted release source: %s", parsedURL.Host)
	}
	return nil
}

func (s *UpdateService) listRollbackVersions(ctx context.Context) ([]RollbackVersionInfo, error) {
	if s.githubClient == nil {
		return nil, infraerrors.ServiceUnavailable("SYSTEM_RELEASE_SOURCE_UNAVAILABLE", "release source is not configured")
	}
	releases, err := s.githubClient.FetchReleases(ctx, githubRepo, 10)
	if err != nil {
		return nil, err
	}

	candidates := make([]RollbackVersionInfo, 0, 3)
	for i := range releases {
		release := &releases[i]
		if err := validateTrustedRelease(release); err != nil {
			continue
		}
		version := normalizeReleaseVersion(release.TagName)
		if version == "" || compareVersions(version, s.currentVersion) >= 0 {
			continue
		}
		artifact, checksum, signature := selectReleaseArtifacts(release, runtime.GOOS, runtime.GOARCH)
		if artifact == nil || checksum == nil {
			continue
		}
		if err := validateDownloadURL(artifact.BrowserDownloadURL); err != nil {
			continue
		}
		if err := validateDownloadURL(checksum.BrowserDownloadURL); err != nil {
			continue
		}
		if signature != nil {
			if err := validateDownloadURL(signature.BrowserDownloadURL); err != nil {
				signature = nil
			}
		}
		candidates = append(candidates, RollbackVersionInfo{
			Version:            version,
			TagName:            release.TagName,
			PublishedAt:        release.PublishedAt,
			HTMLURL:            release.HTMLURL,
			Source:             githubRepo,
			Platform:           runtime.GOOS,
			Arch:               runtime.GOARCH,
			SHA256Available:    true,
			SignatureAvailable: signature != nil,
		})
		if len(candidates) >= 3 {
			break
		}
	}
	return candidates, nil
}

func normalizeReleaseVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if !releaseSemverPattern.MatchString(trimmed) {
		return ""
	}
	return strings.TrimPrefix(trimmed, "v")
}

func validateTrustedRelease(release *GitHubRelease) error {
	if release == nil {
		return infraerrors.ServiceUnavailable("SYSTEM_RELEASE_SOURCE_UNAVAILABLE", "release source is unavailable")
	}
	if normalizeReleaseVersion(release.TagName) == "" {
		return infraerrors.BadRequest("SYSTEM_RELEASE_TAG_INVALID", "release tag must be a final semantic version")
	}
	parsedURL, err := url.Parse(strings.TrimSpace(release.HTMLURL))
	if err != nil {
		return infraerrors.Forbidden("SYSTEM_RELEASE_SOURCE_UNTRUSTED", "release source is not trusted")
	}
	if !isTrustedReleasePageURL(parsedURL) {
		return infraerrors.Forbidden("SYSTEM_RELEASE_SOURCE_UNTRUSTED", "release source is not trusted")
	}
	return nil
}

func selectReleaseArtifacts(release *GitHubRelease, goos, goarch string) (artifact, checksum, signature *GitHubAsset) {
	if release == nil {
		return nil, nil, nil
	}
	archiveName := fmt.Sprintf("%s_%s", goos, goarch)
	for i := range release.Assets {
		asset := &release.Assets[i]
		name := strings.TrimSpace(asset.Name)
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(name, archiveName) && !strings.HasSuffix(lower, ".txt") && !isSignatureAssetName(lower):
			artifact = asset
		case name == "checksums.txt":
			checksum = asset
		case isSignatureAssetName(lower):
			signature = asset
		}
	}
	return artifact, checksum, signature
}

func isSignatureAssetName(name string) bool {
	return strings.HasSuffix(name, ".sig") ||
		strings.HasSuffix(name, ".sigstore") ||
		strings.HasSuffix(name, ".pem") ||
		strings.HasSuffix(name, ".crt") ||
		strings.Contains(name, "signature")
}

func isTrustedReleasePageURL(parsedURL *url.URL) bool {
	if parsedURL == nil || parsedURL.Scheme != "https" || !strings.EqualFold(parsedURL.Host, allowedDownloadHost) {
		return false
	}
	path := strings.Trim(parsedURL.EscapedPath(), "/")
	return strings.HasPrefix(path, githubRepo+"/releases/")
}

func isTrustedReleaseAssetURL(parsedURL *url.URL) bool {
	if parsedURL == nil || parsedURL.Scheme != "https" || !strings.EqualFold(parsedURL.Host, allowedDownloadHost) {
		return false
	}
	path := strings.Trim(parsedURL.EscapedPath(), "/")
	return strings.HasPrefix(path, githubRepo+"/releases/download/")
}
