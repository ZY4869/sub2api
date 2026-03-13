package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	telegramDefaultAPIBaseURL  = "https://api.telegram.org"
	telegramSendLimitPerMinute = 20
	telegramSendWindow         = time.Minute
)

type telegramMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type telegramMessageResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

// TelegramNotifierService sends scheduled test notifications to Telegram.
type TelegramNotifierService struct {
	settingService *SettingService
	client         *http.Client
	apiBaseURL     string
	now            func() time.Time

	mu          sync.Mutex
	recentSends []time.Time
}

func NewTelegramNotifierService(settingService *SettingService) *TelegramNotifierService {
	return &TelegramNotifierService{
		settingService: settingService,
		client:         &http.Client{Timeout: 5 * time.Second},
		apiBaseURL:     telegramDefaultAPIBaseURL,
		now:            time.Now,
	}
}

func (s *TelegramNotifierService) SendNotification(ctx context.Context, message string) error {
	cfg, err := s.resolveConfig(ctx, "", "")
	if err != nil {
		return err
	}
	return s.sendMessage(ctx, cfg.botToken, cfg.chatID, message)
}

func (s *TelegramNotifierService) TestConnection(ctx context.Context, botToken, chatID string) error {
	cfg, err := s.resolveConfig(ctx, botToken, chatID)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Sub2API Telegram connection test succeeded at %s", s.now().Format(time.RFC3339))
	return s.sendMessage(ctx, cfg.botToken, cfg.chatID, message)
}

type resolvedTelegramConfig struct {
	botToken string
	chatID   string
}

func (s *TelegramNotifierService) resolveConfig(ctx context.Context, botToken, chatID string) (*resolvedTelegramConfig, error) {
	botToken = strings.TrimSpace(botToken)
	chatID = strings.TrimSpace(chatID)

	if s.settingService != nil && (botToken == "" || chatID == "") {
		settings, err := s.settingService.GetAllSettings(ctx)
		if err != nil {
			return nil, fmt.Errorf("load telegram settings: %w", err)
		}
		if botToken == "" {
			botToken = strings.TrimSpace(settings.TelegramBotToken)
		}
		if chatID == "" {
			chatID = strings.TrimSpace(settings.TelegramChatID)
		}
	}

	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token is not configured")
	}
	if chatID == "" {
		return nil, fmt.Errorf("telegram chat id is not configured")
	}

	return &resolvedTelegramConfig{botToken: botToken, chatID: chatID}, nil
}

func (s *TelegramNotifierService) sendMessage(ctx context.Context, botToken, chatID, message string) error {
	if err := s.allowSend(); err != nil {
		return err
	}

	payload, err := json.Marshal(telegramMessageRequest{
		ChatID: chatID,
		Text:   strings.TrimSpace(message),
	})
	if err != nil {
		return fmt.Errorf("marshal telegram message: %w", err)
	}

	apiBaseURL := strings.TrimRight(s.apiBaseURL, "/")
	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return fmt.Errorf("read telegram response: %w", err)
	}

	var telegramResp telegramMessageResponse
	if len(body) > 0 {
		_ = json.Unmarshal(body, &telegramResp)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || !telegramResp.OK {
		description := strings.TrimSpace(telegramResp.Description)
		if description == "" {
			description = strings.TrimSpace(string(body))
		}
		if description == "" {
			description = resp.Status
		}
		return fmt.Errorf("telegram send failed: %s", description)
	}

	return nil
}

func (s *TelegramNotifierService) allowSend() error {
	if s == nil {
		return fmt.Errorf("telegram notifier is not available")
	}
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.recentSends[:0]
	for _, ts := range s.recentSends {
		if now.Sub(ts) < telegramSendWindow {
			filtered = append(filtered, ts)
		}
	}
	s.recentSends = filtered

	if len(s.recentSends) >= telegramSendLimitPerMinute {
		logger.LegacyPrintf("service.telegram_notifier", "[TelegramNotifier] rate limit reached, skip send")
		return fmt.Errorf("telegram rate limit exceeded")
	}

	s.recentSends = append(s.recentSends, now)
	return nil
}
