package httputil

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

const (
	requestBodyReadInitCap    = 512
	requestBodyReadMaxInitCap = 1 << 20

	// Security: hard cap the decoded body size to prevent decompression bombs.
	// This limit is applied after decoding Content-Encoding (gzip/deflate/zstd),
	// and is independent from MaxBytesReader which only limits encoded bytes.
	defaultMaxDecodedRequestBodyBytes = int64(64 * 1024 * 1024)
)

var errUnsupportedContentEncoding = errors.New("unsupported content-encoding")

// ReadRequestBodyWithPrealloc reads request body with preallocated buffer based on content length.
func ReadRequestBodyWithPrealloc(req *http.Request) ([]byte, error) {
	return readRequestBodyWithPrealloc(req, defaultMaxDecodedRequestBodyBytes)
}

func readRequestBodyWithPrealloc(req *http.Request, maxDecodedBytes int64) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	capHint := requestBodyReadInitCap
	if req.ContentLength > 0 {
		switch {
		case req.ContentLength < int64(requestBodyReadInitCap):
			capHint = requestBodyReadInitCap
		case req.ContentLength > int64(requestBodyReadMaxInitCap):
			capHint = requestBodyReadMaxInitCap
		default:
			capHint = int(req.ContentLength)
		}
	}

	reader := io.Reader(req.Body)
	encoding := strings.TrimSpace(req.Header.Get("Content-Encoding"))
	normalizedEncoding, hasEncoding, err := normalizeSingleContentEncoding(encoding)
	if err != nil {
		logBodyDecodeFailure(req, "content_encoding_parse_failed", encoding, err)
		return nil, err
	}

	var closer io.Closer
	if hasEncoding {
		decodedReader, decodedCloser, decodeErr := wrapDecodedBodyReader(reader, normalizedEncoding)
		if decodeErr != nil {
			logBodyDecodeFailure(req, "content_encoding_decode_failed", normalizedEncoding, decodeErr)
			return nil, decodeErr
		}
		reader = decodedReader
		closer = decodedCloser
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	// Limit decoded bytes to prevent decompression bombs. Read max+1 bytes so we can
	// detect overflow and return a MaxBytesError compatible with existing handlers.
	if maxDecodedBytes > 0 {
		reader = io.LimitReader(reader, maxDecodedBytes+1)
	}

	buf := bytes.NewBuffer(make([]byte, 0, capHint))
	n, err := io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}

	if maxDecodedBytes > 0 && n > maxDecodedBytes {
		maxErr := &http.MaxBytesError{Limit: maxDecodedBytes}
		logBodyDecodeFailure(req, "decoded_body_too_large", normalizedEncoding, maxErr)
		return nil, maxErr
	}

	// Decode succeeded: clear Content-Encoding so downstream never forwards
	// a decoded body with the original encoding metadata.
	if hasEncoding {
		req.Header.Del("Content-Encoding")
	}

	return buf.Bytes(), nil
}

func normalizeSingleContentEncoding(raw string) (string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return "", false, nil
	}

	parts := strings.Split(raw, ",")
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" || p == "identity" {
			continue
		}
		normalized = append(normalized, p)
	}

	if len(normalized) == 0 {
		return "", false, nil
	}
	if len(normalized) != 1 {
		return "", false, fmt.Errorf("%w: multiple encodings are not supported", errUnsupportedContentEncoding)
	}

	switch normalized[0] {
	case "gzip", "deflate", "zstd":
		return normalized[0], true, nil
	default:
		return "", false, fmt.Errorf("%w: %s", errUnsupportedContentEncoding, normalized[0])
	}
}

func wrapDecodedBodyReader(src io.Reader, encoding string) (io.Reader, io.Closer, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip":
		r, err := gzip.NewReader(src)
		if err != nil {
			return nil, nil, err
		}
		return r, r, nil
	case "deflate":
		// RFC 2616 historically defined "deflate" as zlib-wrapped data, but some
		// clients send raw DEFLATE streams. We must not probe by attempting a zlib
		// decoder directly because it may consume bytes on failure, breaking the
		// fallback. Instead, sniff the zlib header via Peek.
		br := bufio.NewReader(src)
		if header, err := br.Peek(2); err == nil && looksLikeZlibHeader(header[0], header[1]) {
			zr, err := zlib.NewReader(br)
			if err != nil {
				return nil, nil, err
			}
			return zr, zr, nil
		}
		fr := flate.NewReader(br)
		return fr, fr, nil
	case "zstd":
		dec, err := zstd.NewReader(src)
		if err != nil {
			return nil, nil, err
		}
		// klauspost/compress/zstd.Decoder has Close() with no error return.
		return dec, closerFunc(func() error {
			dec.Close()
			return nil
		}), nil
	default:
		return nil, nil, fmt.Errorf("%w: %s", errUnsupportedContentEncoding, strings.TrimSpace(encoding))
	}
}

type closerFunc func() error

func (c closerFunc) Close() error { return c() }

func looksLikeZlibHeader(cmf, flg byte) bool {
	// zlib header: (CMF<<8 + FLG) % 31 == 0
	// CMF lower 4 bits must be 8 (deflate), and CINFO must be <= 7.
	if (cmf & 0x0F) != 8 {
		return false
	}
	if (cmf>>4)&0x0F > 7 {
		return false
	}
	return (int(cmf)<<8+int(flg))%31 == 0
}

func logBodyDecodeFailure(req *http.Request, event string, encoding string, err error) {
	if req == nil || err == nil {
		return
	}
	logger.FromContext(req.Context()).Warn(
		"request_body_decode_failed",
		zap.String("event", strings.TrimSpace(event)),
		zap.String("content_encoding", strings.TrimSpace(encoding)),
		zap.Error(err),
	)
}
