package service

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type openAICodexAppServerConn struct {
	ctx       context.Context
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	scanner   *bufio.Scanner
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[int64]chan openAICodexJSONRPCResponse
	nextID    int64
	done      chan struct{}
	closeOnce sync.Once
	stderrBuf *cappedStringBuffer
	auth      OpenAICodexAppServerAuthTokens
}

type openAICodexJSONRPCRequest struct {
	Method string `json:"method"`
	ID     int64  `json:"id,omitempty"`
	Params any    `json:"params,omitempty"`
}

type openAICodexJSONRPCResponse struct {
	ID     int64                    `json:"id"`
	Result json.RawMessage          `json:"result,omitempty"`
	Error  *openAICodexJSONRPCError `json:"error,omitempty"`
}

type openAICodexJSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *openAICodexAppServerConn) initialize(auth OpenAICodexAppServerAuthTokens) error {
	if _, err := c.call("initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "sub2api",
			"title":   "Sub2API",
			"version": openAICodexAppServerClientVersion,
		},
		"capabilities": map[string]any{
			"experimentalApi": true,
		},
	}); err != nil {
		return err
	}
	if err := c.notify("initialized", map[string]any{}); err != nil {
		return err
	}
	_, err := c.call("account/login/start", map[string]any{
		"type":             "chatgptAuthTokens",
		"accessToken":      strings.TrimSpace(auth.AccessToken),
		"chatgptAccountId": strings.TrimSpace(auth.ChatGPTAccountID),
		"chatgptPlanType":  strings.TrimSpace(auth.ChatGPTPlanType),
	})
	return err
}

func (c *openAICodexAppServerConn) call(method string, params any) (json.RawMessage, error) {
	id := c.nextRequestID()
	ch := make(chan openAICodexJSONRPCResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	if err := c.write(openAICodexJSONRPCRequest{Method: method, ID: id, Params: params}); err != nil {
		c.removePending(id)
		return nil, err
	}

	select {
	case <-c.ctx.Done():
		c.removePending(id)
		return nil, openAICodexAppServerTimeoutError(c.ctx.Err())
	case <-c.done:
		c.removePending(id)
		return nil, c.processExitError()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, openAICodexJSONRPCApplicationError(method, resp.Error)
		}
		return resp.Result, nil
	}
}

func (c *openAICodexAppServerConn) notify(method string, params any) error {
	return c.write(openAICodexJSONRPCRequest{Method: method, Params: params})
}

func (c *openAICodexAppServerConn) write(req openAICodexJSONRPCRequest) error {
	return c.writeRaw(req)
}

func (c *openAICodexAppServerConn) writeRaw(value any) error {
	line, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := c.stdin.Write(append(line, '\n')); err != nil {
		return sanitizeOpenAICodexAppServerError("OPENAI_CODEX_APP_SERVER_WRITE_FAILED", err)
	}
	return nil
}

func (c *openAICodexAppServerConn) close() {
	c.closeOnce.Do(func() {
		_ = c.stdin.Close()
		c.cancel()
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		if c.cmd != nil {
			_ = c.cmd.Wait()
		}
	})
}

func (c *openAICodexAppServerConn) processExitError() error {
	return infraerrors.ServiceUnavailable("OPENAI_CODEX_APP_SERVER_UNAVAILABLE", "Codex app-server 不可用")
}
