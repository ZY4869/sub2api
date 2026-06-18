package service

import (
	"encoding/json"
	"strings"
	"sync"
)

func (c *openAICodexAppServerConn) readLoop() {
	defer close(c.done)
	for c.scanner.Scan() {
		line := strings.TrimSpace(c.scanner.Text())
		if line == "" {
			continue
		}
		var envelope struct {
			ID     *int64           `json:"id"`
			Method string           `json:"method"`
			Result *json.RawMessage `json:"result"`
			Error  *json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if envelope.ID != nil && envelope.Method == "account/chatgptAuthTokens/refresh" {
			_ = c.writeRaw(map[string]any{
				"id": *envelope.ID,
				"result": map[string]any{
					"accessToken":      strings.TrimSpace(c.auth.AccessToken),
					"chatgptAccountId": strings.TrimSpace(c.auth.ChatGPTAccountID),
					"chatgptPlanType":  strings.TrimSpace(c.auth.ChatGPTPlanType),
				},
			})
			continue
		}
		if envelope.ID == nil || (envelope.Result == nil && envelope.Error == nil) {
			continue
		}
		var resp openAICodexJSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		c.pendingMu.Lock()
		ch := c.pending[resp.ID]
		delete(c.pending, resp.ID)
		c.pendingMu.Unlock()
		if ch != nil {
			ch <- resp
		}
	}
}

func (c *openAICodexAppServerConn) nextRequestID() int64 {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	id := c.nextID
	c.nextID++
	return id
}

func (c *openAICodexAppServerConn) removePending(id int64) {
	c.pendingMu.Lock()
	delete(c.pending, id)
	c.pendingMu.Unlock()
}

type cappedStringBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func (b *cappedStringBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if len(p) > remaining {
			b.data = append(b.data, p[:remaining]...)
		} else {
			b.data = append(b.data, p...)
		}
	}
	return len(p), nil
}

func (b *cappedStringBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
