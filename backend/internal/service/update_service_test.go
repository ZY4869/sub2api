package service

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type updateReleaseClientStub struct {
	latest   *GitHubRelease
	releases []GitHubRelease
}

func (s updateReleaseClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	return s.latest, nil
}

func (s updateReleaseClientStub) FetchReleases(context.Context, string, int) ([]GitHubRelease, error) {
	return s.releases, nil
}

func (s updateReleaseClientStub) DownloadFile(context.Context, string, string, int64) error {
	return nil
}

func (s updateReleaseClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, nil
}

type updateCacheStub struct {
	data string
	ttl  time.Duration
}

func (c *updateCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if c.data == "" {
		return "", infraerrors.NotFound("CACHE_MISS", "cache miss")
	}
	return c.data, nil
}

func (c *updateCacheStub) SetUpdateInfo(_ context.Context, data string, ttl time.Duration) error {
	c.data = data
	c.ttl = ttl
	return nil
}

func TestUpdateService_CheckUpdateRollbackVersionsTrustedOnly(t *testing.T) {
	client := updateReleaseClientStub{
		latest: trustedRelease("v0.1.380", false),
		releases: []GitHubRelease{
			*trustedRelease("v0.1.380", false),
			*trustedRelease("v0.1.378", true),
			*untrustedRelease("v0.1.377"),
			*trustedRelease("v0.1.376", true),
			*trustedRelease("v0.1.375", true),
			*trustedRelease("v0.1.374", true),
		},
	}
	svc := NewUpdateService(nil, client, "0.1.379", "release")

	info, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Len(t, info.RollbackVersions, 3)
	require.Equal(t, []string{"0.1.378", "0.1.376", "0.1.375"}, rollbackVersionIDs(info.RollbackVersions))
	for _, candidate := range info.RollbackVersions {
		require.Equal(t, githubRepo, candidate.Source)
		require.True(t, candidate.SHA256Available)
		require.True(t, candidate.SignatureAvailable)
	}
}

func TestUpdateService_CheckUpdateCachePreservesRollbackVersions(t *testing.T) {
	cache := &updateCacheStub{}
	client := updateReleaseClientStub{
		latest: trustedRelease("v0.1.380", false),
		releases: []GitHubRelease{
			*trustedRelease("v0.1.378", true),
		},
	}
	svc := NewUpdateService(cache, client, "0.1.379", "release")

	fresh, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)
	require.Len(t, fresh.RollbackVersions, 1)

	cached, err := svc.CheckUpdate(context.Background(), false)
	require.NoError(t, err)
	require.True(t, cached.Cached)
	require.Equal(t, rollbackVersionIDs(fresh.RollbackVersions), rollbackVersionIDs(cached.RollbackVersions))
	require.Equal(t, time.Duration(updateCacheTTL)*time.Second, cache.ttl)
}

func TestUpdateService_RollbackToVersionRequiresSignature(t *testing.T) {
	client := updateReleaseClientStub{
		releases: []GitHubRelease{*trustedRelease("v0.1.378", false)},
	}
	svc := NewUpdateService(nil, client, "0.1.379", "release")

	err := svc.RollbackToVersion(context.Background(), "v0.1.378")
	require.Error(t, err)
	require.Equal(t, "SYSTEM_ROLLBACK_SIGNATURE_REQUIRED", infraerrors.Reason(err))
	require.Equal(t, 403, infraerrors.Code(err))
}

func TestUpdateService_RollbackToVersionRejectsUnknownTarget(t *testing.T) {
	client := updateReleaseClientStub{
		releases: []GitHubRelease{*trustedRelease("v0.1.378", true)},
	}
	svc := NewUpdateService(nil, client, "0.1.379", "release")

	err := svc.RollbackToVersion(context.Background(), "0.1.100")
	require.Error(t, err)
	require.Equal(t, "SYSTEM_ROLLBACK_TARGET_NOT_FOUND", infraerrors.Reason(err))
	require.Equal(t, 404, infraerrors.Code(err))
}

func TestValidateDownloadURLRestrictsReleaseSource(t *testing.T) {
	require.NoError(t, validateDownloadURL("https://github.com/ZY4869/sub2api/releases/download/v0.1.379/sub2api_0.1.379_linux_amd64.tar.gz"))
	require.Error(t, validateDownloadURL("https://github.com/Wei-Shaw/sub2api/releases/download/v0.1.150/sub2api_0.1.150_linux_amd64.tar.gz"))
	require.Error(t, validateDownloadURL("https://objects.githubusercontent.com/github-production-release-asset/file"))
	require.Error(t, validateDownloadURL("https://hub.docker.com/r/wei-shaw/sub2api/tags"))
}

func trustedRelease(tag string, withSignature bool) *GitHubRelease {
	version := normalizeReleaseVersion(tag)
	assets := []GitHubAsset{
		{
			Name:               "sub2api_" + version + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
			BrowserDownloadURL: "https://github.com/ZY4869/sub2api/releases/download/" + tag + "/sub2api_" + version + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
			Size:               1024,
		},
		{
			Name:               "checksums.txt",
			BrowserDownloadURL: "https://github.com/ZY4869/sub2api/releases/download/" + tag + "/checksums.txt",
			Size:               128,
		},
	}
	if withSignature {
		assets = append(assets, GitHubAsset{
			Name:               "checksums.txt.sig",
			BrowserDownloadURL: "https://github.com/ZY4869/sub2api/releases/download/" + tag + "/checksums.txt.sig",
			Size:               128,
		})
	}
	return &GitHubRelease{
		TagName:     tag,
		Name:        "Sub2API " + version,
		PublishedAt: "2026-07-01T00:00:00Z",
		HTMLURL:     "https://github.com/ZY4869/sub2api/releases/tag/" + tag,
		Assets:      assets,
	}
}

func untrustedRelease(tag string) *GitHubRelease {
	release := trustedRelease(tag, true)
	release.HTMLURL = "https://github.com/Wei-Shaw/sub2api/releases/tag/" + tag
	for i := range release.Assets {
		release.Assets[i].BrowserDownloadURL = strings.Replace(release.Assets[i].BrowserDownloadURL, "ZY4869", "Wei-Shaw", 1)
	}
	return release
}

func rollbackVersionIDs(items []RollbackVersionInfo) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Version)
	}
	return out
}
