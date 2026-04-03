package service

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const opsRequestTraceCipherPrefix = "gtr1"

func deriveOpsRequestTraceKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("request trace encryption key is empty")
	}

	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) > 0 {
		if len(decoded) == 32 {
			return decoded, nil
		}
		sum := sha256.Sum256(decoded)
		return sum[:], nil
	}

	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func encryptOpsRequestTracePayload(key string, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	derivedKey, err := deriveOpsRequestTraceKey(key)
	if err != nil {
		return nil, err
	}
	compressed, err := gzipCompressBytes(plaintext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return nil, err
	}

	sealed := gcm.Seal(nil, nonce, compressed, nil)
	out := make([]byte, 0, len(opsRequestTraceCipherPrefix)+len(nonce)+len(sealed))
	out = append(out, []byte(opsRequestTraceCipherPrefix)...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

func decryptOpsRequestTracePayload(key string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	derivedKey, err := deriveOpsRequestTraceKey(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < len(opsRequestTraceCipherPrefix) || string(ciphertext[:len(opsRequestTraceCipherPrefix)]) != opsRequestTraceCipherPrefix {
		return nil, fmt.Errorf("unsupported request trace payload format")
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	payload := ciphertext[len(opsRequestTraceCipherPrefix):]
	if len(payload) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid request trace payload")
	}

	nonce := payload[:gcm.NonceSize()]
	sealed := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, err
	}
	return gzipDecompressBytes(plain)
}

func gzipCompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gzipDecompressBytes(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	plain, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return plain, nil
}
