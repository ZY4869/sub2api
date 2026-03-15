package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

const soraImageInputMaxBytes = 20 << 20
const soraImageInputMaxRedirects = 3
const soraImageInputTimeout = 20 * time.Second
const soraVideoInputMaxBytes = 200 << 20
const soraVideoInputMaxRedirects = 3
const soraVideoInputTimeout = 60 * time.Second

func decodeSoraImageInput(ctx context.Context, input string) ([]byte, string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, "", errors.New("empty image input")
	}
	if strings.HasPrefix(raw, "data:") {
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 {
			return nil, "", errors.New("invalid data url")
		}
		meta := parts[0]
		payload := parts[1]
		decoded, err := decodeBase64WithLimit(payload, soraImageInputMaxBytes)
		if err != nil {
			return nil, "", err
		}
		ext := ""
		if strings.HasPrefix(meta, "data:") {
			metaParts := strings.SplitN(meta[5:], ";", 2)
			if len(metaParts) > 0 {
				if exts, err := mime.ExtensionsByType(metaParts[0]); err == nil && len(exts) > 0 {
					ext = exts[0]
				}
			}
		}
		filename := "image" + ext
		return decoded, filename, nil
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return downloadSoraImageInput(ctx, raw)
	}
	decoded, err := decodeBase64WithLimit(raw, soraImageInputMaxBytes)
	if err != nil {
		return nil, "", errors.New("invalid base64 image")
	}
	return decoded, "image.png", nil
}

func decodeSoraVideoInput(ctx context.Context, input string) ([]byte, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, errors.New("empty video input")
	}
	if strings.HasPrefix(raw, "data:") {
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid video data url")
		}
		decoded, err := decodeBase64WithLimit(parts[1], soraVideoInputMaxBytes)
		if err != nil {
			return nil, errors.New("invalid base64 video")
		}
		if len(decoded) == 0 {
			return nil, errors.New("empty video data")
		}
		return decoded, nil
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return downloadSoraVideoInput(ctx, raw)
	}
	decoded, err := decodeBase64WithLimit(raw, soraVideoInputMaxBytes)
	if err != nil {
		return nil, errors.New("invalid base64 video")
	}
	if len(decoded) == 0 {
		return nil, errors.New("empty video data")
	}
	return decoded, nil
}

func downloadSoraImageInput(ctx context.Context, rawURL string) ([]byte, string, error) {
	parsed, err := validateSoraRemoteURL(rawURL)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{
		Timeout: soraImageInputTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= soraImageInputMaxRedirects {
				return errors.New("too many redirects")
			}
			return validateSoraRemoteURLValue(req.URL)
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image failed: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, soraImageInputMaxBytes))
	if err != nil {
		return nil, "", err
	}
	ext := fileExtFromURL(parsed.String())
	if ext == "" {
		ext = fileExtFromContentType(resp.Header.Get("Content-Type"))
	}
	filename := "image" + ext
	return data, filename, nil
}

func downloadSoraVideoInput(ctx context.Context, rawURL string) ([]byte, error) {
	parsed, err := validateSoraRemoteURL(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: soraVideoInputTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= soraVideoInputMaxRedirects {
				return errors.New("too many redirects")
			}
			return validateSoraRemoteURLValue(req.URL)
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download video failed: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, soraVideoInputMaxBytes))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("empty video content")
	}
	return data, nil
}

func decodeBase64WithLimit(encoded string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("invalid max bytes limit")
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	limited := io.LimitReader(decoder, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("input exceeds %d bytes limit", maxBytes)
	}
	return data, nil
}
