package admin

import (
	"errors"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type TestTelegramRequest struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

func (h *SettingHandler) TestTelegramConnection(c *gin.Context) {
	var req TestTelegramRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if h.telegramNotifier == nil {
		response.Error(c, 500, "Telegram notifier is not available")
		return
	}

	if err := h.telegramNotifier.TestConnection(
		c.Request.Context(),
		strings.TrimSpace(req.BotToken),
		strings.TrimSpace(req.ChatID),
	); err != nil {
		response.BadRequest(c, "Telegram connection test failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Telegram connection successful"})
}
