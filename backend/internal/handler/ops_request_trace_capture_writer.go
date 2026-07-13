package handler

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type opsRequestTraceCaptureWriter struct {
	gin.ResponseWriter
	limit     int
	total     int
	truncated bool
	buf       bytes.Buffer
}

var (
	opsRequestTraceWriterPool = sync.Pool{
		New: func() any {
			return &opsRequestTraceCaptureWriter{limit: opsRequestTraceBodyLimit}
		},
	}
)

func acquireOpsRequestTraceCaptureWriter(rw gin.ResponseWriter) *opsRequestTraceCaptureWriter {
	writer, ok := opsRequestTraceWriterPool.Get().(*opsRequestTraceCaptureWriter)
	if !ok || writer == nil {
		writer = &opsRequestTraceCaptureWriter{}
	}
	writer.ResponseWriter = rw
	writer.limit = opsRequestTraceBodyLimit
	writer.total = 0
	writer.truncated = false
	writer.buf.Reset()
	return writer
}

func releaseOpsRequestTraceCaptureWriter(writer *opsRequestTraceCaptureWriter) {
	if writer == nil {
		return
	}
	writer.ResponseWriter = nil
	writer.limit = opsRequestTraceBodyLimit
	writer.total = 0
	writer.truncated = false
	writer.buf.Reset()
	opsRequestTraceWriterPool.Put(writer)
}

func (w *opsRequestTraceCaptureWriter) Header() http.Header {
	if w == nil {
		return http.Header{}
	}
	return opsWriterHeader(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) Status() int {
	if w == nil {
		return http.StatusOK
	}
	return opsWriterStatus(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) Size() int {
	if w == nil {
		return 0
	}
	return opsWriterSize(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) Written() bool {
	return w != nil && opsWriterWritten(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) WriteHeaderNow() {
	if w != nil {
		opsWriterWriteHeaderNow(w.ResponseWriter)
	}
}

func (w *opsRequestTraceCaptureWriter) WriteHeader(code int) {
	if w != nil {
		opsWriterWriteHeader(w.ResponseWriter, code)
	}
}

func (w *opsRequestTraceCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	if w == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return opsWriterWrite(w.ResponseWriter, data)
}

func (w *opsRequestTraceCaptureWriter) WriteString(value string) (int, error) {
	w.capture([]byte(value))
	if w == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return opsWriterWriteString(w.ResponseWriter, value)
}

func (w *opsRequestTraceCaptureWriter) capture(data []byte) {
	if w == nil || len(data) == 0 {
		return
	}
	w.total += len(data)
	if w.limit <= 0 || w.buf.Len() >= w.limit {
		w.truncated = true
		return
	}
	remaining := w.limit - w.buf.Len()
	if len(data) > remaining {
		_, _ = w.buf.Write(data[:remaining])
		w.truncated = true
		return
	}
	_, _ = w.buf.Write(data)
}

func (w *opsRequestTraceCaptureWriter) BytesCopy() []byte {
	if w == nil || w.buf.Len() == 0 {
		return nil
	}
	return append([]byte(nil), w.buf.Bytes()...)
}

func (w *opsRequestTraceCaptureWriter) Flush() {
	if w != nil {
		opsWriterFlush(w.ResponseWriter)
	}
}

func (w *opsRequestTraceCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w == nil {
		return nil, nil, errOpsCaptureWriterReleased
	}
	return opsWriterHijack(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) CloseNotify() <-chan bool {
	if w == nil {
		return opsWriterCloseNotify(nil)
	}
	return opsWriterCloseNotify(w.ResponseWriter)
}

func (w *opsRequestTraceCaptureWriter) Pusher() http.Pusher {
	if w == nil {
		return nil
	}
	return opsWriterPusher(w.ResponseWriter)
}
