package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

const deepSeekInternalUserIDPrefix = "sub2api_"

func (s *OpenAIGatewayService) injectDeepSeekOpenAIUserID(c *gin.Context, account *Account, body []byte) ([]byte, bool, error) {
	userID := s.deriveDeepSeekInternalUserID(c, account)
	if userID == "" {
		return body, false, nil
	}
	updated, err := sjson.SetBytes(body, "user_id", userID)
	if err != nil {
		return nil, false, fmt.Errorf("inject deepseek user_id: %w", err)
	}
	return updated, true, nil
}

func (s *GatewayService) injectDeepSeekAnthropicUserID(c *gin.Context, account *Account, body []byte) ([]byte, bool, error) {
	userID := s.deriveDeepSeekInternalUserID(c, account)
	if userID == "" {
		return body, false, nil
	}
	updated, err := sjson.SetBytes(body, "metadata.user_id", userID)
	if err != nil {
		return nil, false, fmt.Errorf("inject deepseek metadata.user_id: %w", err)
	}
	return updated, true, nil
}

func (s *OpenAIGatewayService) deriveDeepSeekInternalUserID(c *gin.Context, account *Account) string {
	salt := ""
	if s != nil && s.cfg != nil {
		salt = s.cfg.JWT.Secret
	}
	return deriveDeepSeekInternalUserID(c, account, salt)
}

func (s *GatewayService) deriveDeepSeekInternalUserID(c *gin.Context, account *Account) string {
	salt := ""
	if s != nil && s.cfg != nil {
		salt = s.cfg.JWT.Secret
	}
	return deriveDeepSeekInternalUserID(c, account, salt)
}

func deriveDeepSeekInternalUserID(c *gin.Context, account *Account, salt string) string {
	if account == nil || RoutingPlatformForAccount(account) != PlatformDeepSeek {
		return ""
	}
	apiKey := apiKeyFromGinContext(c)
	userID := int64(0)
	apiKeyID := int64(0)
	if apiKey != nil {
		userID = apiKey.UserID
		apiKeyID = apiKey.ID
	}
	if userID <= 0 && apiKey != nil && apiKey.User != nil {
		userID = apiKey.User.ID
	}
	if userID <= 0 && apiKeyID <= 0 {
		return ""
	}
	if strings.TrimSpace(salt) == "" {
		salt = "sub2api-deepseek-user-id"
	}
	mac := hmac.New(sha256.New, []byte(salt))
	_, _ = mac.Write([]byte(fmt.Sprintf("user:%d|api_key:%d|account:%d", userID, apiKeyID, account.ID)))
	return deepSeekInternalUserIDPrefix + hex.EncodeToString(mac.Sum(nil))[:40]
}

func apiKeyFromGinContext(c *gin.Context) *APIKey {
	if c == nil {
		return nil
	}
	value, ok := c.Get("api_key")
	if !ok || value == nil {
		return nil
	}
	apiKey, ok := value.(*APIKey)
	if !ok {
		return nil
	}
	return apiKey
}
