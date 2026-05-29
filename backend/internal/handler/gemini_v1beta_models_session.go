package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// geminiCLITmpDirRegex 用于从 Gemini CLI 请求体中提取 tmp 目录的哈希值
// 匹配格式: /Users/xxx/.gemini/tmp/[64位十六进制哈希]
var geminiCLITmpDirRegex = regexp.MustCompile(`/\.gemini/tmp/([A-Fa-f0-9]{64})`)

func extractGeminiCLISessionHash(c *gin.Context, body []byte) string {
	match := geminiCLITmpDirRegex.FindSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	tmpDirHash := string(match[1])

	privilegedUserID := strings.TrimSpace(c.GetHeader("x-gemini-api-privileged-user-id"))
	if privilegedUserID != "" {
		combined := privilegedUserID + ":" + tmpDirHash
		hash := sha256.Sum256([]byte(combined))
		return hex.EncodeToString(hash[:])
	}
	return tmpDirHash
}

func truncateDigestChain(chain string) string {
	if len(chain) <= 50 {
		return chain
	}
	return chain[:50] + "..."
}

func safeShortPrefix(value string, n int) string {
	if n <= 0 || len(value) <= n {
		return value
	}
	return value[:n]
}

func derefGroupID(groupID *int64) int64 {
	if groupID == nil {
		return 0
	}
	return *groupID
}
