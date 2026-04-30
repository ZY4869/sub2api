package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestReadRequestBodyWithPrealloc_DecodeGzip(t *testing.T) {
	payload := []byte(`{"model":"gpt-5","input":"hello"}`)
	compressed := gzipCompress(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "gzip")

	body, err := ReadRequestBodyWithPrealloc(req)
	require.NoError(t, err)
	require.Equal(t, payload, body)
	require.Empty(t, req.Header.Get("Content-Encoding"))
}

func TestReadRequestBodyWithPrealloc_DecodeDeflate_Zlib(t *testing.T) {
	payload := []byte(`{"model":"gpt-5","input":"hello"}`)
	compressed := zlibCompress(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "deflate")

	body, err := ReadRequestBodyWithPrealloc(req)
	require.NoError(t, err)
	require.Equal(t, payload, body)
	require.Empty(t, req.Header.Get("Content-Encoding"))
}

func TestReadRequestBodyWithPrealloc_DecodeDeflate_Raw(t *testing.T) {
	payload := []byte(`{"model":"gpt-5","input":"hello"}`)
	compressed := flateCompress(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "deflate")

	body, err := ReadRequestBodyWithPrealloc(req)
	require.NoError(t, err)
	require.Equal(t, payload, body)
	require.Empty(t, req.Header.Get("Content-Encoding"))
}

func TestReadRequestBodyWithPrealloc_DecodeZstd(t *testing.T) {
	payload := []byte(`{"model":"gpt-5","input":"hello"}`)
	compressed := zstdCompress(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "zstd")

	body, err := ReadRequestBodyWithPrealloc(req)
	require.NoError(t, err)
	require.Equal(t, payload, body)
	require.Empty(t, req.Header.Get("Content-Encoding"))
}

func TestReadRequestBodyWithPrealloc_UnsupportedEncoding(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Encoding", "br")

	_, err := ReadRequestBodyWithPrealloc(req)
	require.Error(t, err)
	require.ErrorIs(t, err, errUnsupportedContentEncoding)
}

func TestReadRequestBodyWithPrealloc_UnsupportedEncoding_Multiple(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte("x")))
	req.Header.Set("Content-Encoding", "gzip, deflate")

	_, err := ReadRequestBodyWithPrealloc(req)
	require.Error(t, err)
	require.ErrorIs(t, err, errUnsupportedContentEncoding)
}

func TestReadRequestBodyWithPrealloc_DecodedBodyHardLimit(t *testing.T) {
	payload := []byte("hello world") // 11 bytes
	compressed := gzipCompress(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "gzip")

	_, err := readRequestBodyWithPrealloc(req, 10)
	require.Error(t, err)

	var maxErr *http.MaxBytesError
	require.ErrorAs(t, err, &maxErr)
	require.Equal(t, int64(10), maxErr.Limit)
}

func gzipCompress(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func zlibCompress(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func flateCompress(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func zstdCompress(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}
