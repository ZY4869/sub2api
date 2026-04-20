package repository

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountEntityToService_CanonicalizesBaiduPlatform(t *testing.T) {
	account := accountEntityToService(&dbent.Account{
		ID:       1,
		Name:     "legacy-baidu",
		Platform: "baidu",
	})
	require.NotNil(t, account)
	require.Equal(t, service.PlatformBaiduDocumentAI, account.Platform)
}

func TestGroupEntityToService_CanonicalizesBaiduPlatform(t *testing.T) {
	group := groupEntityToService(&dbent.Group{
		ID:       2,
		Name:     "legacy-baidu-group",
		Platform: "baidu",
	})
	require.NotNil(t, group)
	require.Equal(t, service.PlatformBaiduDocumentAI, group.Platform)
}
