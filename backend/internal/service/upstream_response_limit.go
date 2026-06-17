package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/klauspost/compress/zstd"
)

var ErrUpstreamResponseBodyTooLarge = errors.New("upstream response body too large")

const defaultUpstreamResponseReadMaxBytes int64 = 1024 * 1024 * 1024

func resolveUpstreamResponseReadLimit(cfg *config.Config) int64 {
	if cfg != nil && cfg.Gateway.UpstreamResponseReadMaxBytes > 0 {
		return cfg.Gateway.UpstreamResponseReadMaxBytes
	}
	return defaultUpstreamResponseReadMaxBytes
}

func readUpstreamResponseBodyLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	if reader == nil {
		return nil, errors.New("response body is nil")
	}
	if maxBytes <= 0 {
		maxBytes = defaultUpstreamResponseReadMaxBytes
	}

	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%w: limit=%d", ErrUpstreamResponseBodyTooLarge, maxBytes)
	}
	return body, nil
}

func upstreamResponseBodyReader(resp *http.Response) (io.Reader, func(), error) {
	if resp == nil || resp.Body == nil {
		return nil, nil, errors.New("response body is nil")
	}
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if encoding == "" || encoding == "identity" {
		return resp.Body, func() {}, nil
	}
	switch encoding {
	case "zstd", "zstandard":
		decoder, err := zstd.NewReader(resp.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("decode zstd upstream response: %w", err)
		}
		return decoder, decoder.Close, nil
	default:
		return resp.Body, func() {}, nil
	}
}

func readUpstreamResponseBodyLimitedFromResponse(resp *http.Response, maxBytes int64) ([]byte, error) {
	reader, cleanup, err := upstreamResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return readUpstreamResponseBodyLimited(reader, maxBytes)
}
