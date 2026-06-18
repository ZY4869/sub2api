package service

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const openAICodexAppServerClientVersion = "0.1.0"

type OpenAICodexAppServerClient struct {
	bin                 string
	timeout             time.Duration
	capabilityMu        sync.Mutex
	resetConsumeSupport *bool
}

func NewOpenAICodexAppServerClient(cfg *config.Config) *OpenAICodexAppServerClient {
	bin := "codex"
	timeout := 15 * time.Second
	if cfg != nil {
		if strings.TrimSpace(cfg.OpenAICodex.AppServerBin) != "" {
			bin = strings.TrimSpace(cfg.OpenAICodex.AppServerBin)
		}
		if cfg.OpenAICodex.AppServerTimeoutSeconds > 0 {
			timeout = time.Duration(cfg.OpenAICodex.AppServerTimeoutSeconds) * time.Second
		}
	}
	return &OpenAICodexAppServerClient{bin: bin, timeout: timeout}
}

func (c *OpenAICodexAppServerClient) ReadRateLimits(ctx context.Context, auth OpenAICodexAppServerAuthTokens) (*OpenAICodexAppServerRateLimitsSnapshot, error) {
	conn, err := c.open(ctx, auth)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	raw, err := conn.call("account/rateLimits/read", nil)
	if err != nil {
		return nil, err
	}
	return parseOpenAICodexRateLimitsResult(raw, time.Now())
}

func (c *OpenAICodexAppServerClient) ConsumeResetCredit(ctx context.Context, auth OpenAICodexAppServerAuthTokens, idempotencyKey string) (*OpenAICodexAppServerConsumeResult, error) {
	supported, known, err := c.supportsResetCreditConsume(ctx)
	if err != nil {
		return nil, err
	}
	if known && !supported {
		return nil, openAICodexResetCreditsUnsupportedError()
	}

	conn, err := c.open(ctx, auth)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	raw, err := conn.call("account/rateLimitResetCredit/consume", map[string]any{
		"idempotencyKey": strings.TrimSpace(idempotencyKey),
	})
	if err != nil {
		return nil, err
	}
	status, err := parseOpenAICodexResetCreditConsumeStatus(raw)
	if err != nil {
		return nil, err
	}

	latestRaw, err := conn.call("account/rateLimits/read", nil)
	if err != nil {
		return nil, err
	}
	snapshot, err := parseOpenAICodexRateLimitsResult(latestRaw, time.Now())
	if err != nil {
		return nil, err
	}
	return &OpenAICodexAppServerConsumeResult{Status: status, Snapshot: snapshot}, nil
}

func (c *OpenAICodexAppServerClient) open(ctx context.Context, auth OpenAICodexAppServerAuthTokens) (*openAICodexAppServerConn, error) {
	if c == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_NOT_CONFIGURED", "Codex app-server 未配置")
	}
	bin := strings.TrimSpace(c.bin)
	if bin == "" {
		return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_NOT_CONFIGURED", "Codex app-server 未配置")
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(callCtx, bin, "app-server")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_START_FAILED", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_START_FAILED", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_START_FAILED", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		if errors.Is(err, exec.ErrNotFound) {
			return nil, infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_NOT_CONFIGURED", "Codex app-server 未配置或不可执行")
		}
		return nil, sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_START_FAILED", err)
	}

	conn := &openAICodexAppServerConn{
		ctx:       callCtx,
		cancel:    cancel,
		cmd:       cmd,
		stdin:     stdin,
		scanner:   bufio.NewScanner(stdout),
		pending:   make(map[int64]chan openAICodexJSONRPCResponse),
		nextID:    1,
		done:      make(chan struct{}),
		stderrBuf: &cappedStringBuffer{limit: 2048},
		auth:      auth,
	}
	conn.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	go conn.readLoop()
	go func() {
		_, _ = io.Copy(conn.stderrBuf, stderr)
	}()

	if err := conn.initialize(auth); err != nil {
		conn.close()
		return nil, err
	}
	return conn, nil
}
