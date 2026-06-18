package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errResetCreditConsumeSchemaFound = errors.New("reset credit consume schema found")

func (c *OpenAICodexAppServerClient) supportsResetCreditConsume(ctx context.Context) (supported bool, known bool, err error) {
	if c == nil {
		return false, true, openAICodexResetCreditsUnsupportedError()
	}
	c.capabilityMu.Lock()
	if c.resetConsumeSupport != nil {
		defer c.capabilityMu.Unlock()
		return *c.resetConsumeSupport, true, nil
	}
	c.capabilityMu.Unlock()

	supported, known = c.probeResetCreditConsumeSchema(ctx)
	if known {
		c.capabilityMu.Lock()
		c.resetConsumeSupport = &supported
		c.capabilityMu.Unlock()
	}
	return supported, known, nil
}

func (c *OpenAICodexAppServerClient) probeResetCreditConsumeSchema(ctx context.Context) (bool, bool) {
	bin := strings.TrimSpace(c.bin)
	if bin == "" {
		return false, true
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	outDir, err := os.MkdirTemp("", "sub2api-codex-schema-*")
	if err != nil {
		return false, false
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	cmd := exec.CommandContext(callCtx, bin, "app-server", "generate-json-schema", "--experimental", "--out", outDir)
	if err := cmd.Run(); err != nil {
		return false, false
	}

	return schemaDirContainsResetCreditConsume(outDir)
}

func schemaDirContainsResetCreditConsume(outDir string) (bool, bool) {
	var sawSchema bool
	err := filepath.WalkDir(outDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		sawSchema = true
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(content), "account/rateLimitResetCredit/consume") {
			return errResetCreditConsumeSchemaFound
		}
		return nil
	})
	if err == errResetCreditConsumeSchemaFound {
		return true, true
	}
	return false, sawSchema
}
