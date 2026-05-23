//go:build unit

package service

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDeriveDeepSeekInternalUserIDStableAndNonPII(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set("api_key", &APIKey{ID: 12, UserID: 34, User: &User{ID: 34, Email: "person@example.com"}})
	account := &Account{ID: 56, Platform: PlatformDeepSeek}

	first := deriveDeepSeekInternalUserID(c, account, "secret")
	second := deriveDeepSeekInternalUserID(c, account, "secret")

	require.Equal(t, first, second)
	require.Regexp(t, `^sub2api_[a-f0-9]{40}$`, first)
	require.NotContains(t, first, "example")
	require.NotContains(t, first, "person")
}
