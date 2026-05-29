package handler

import (
	"bytes"
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

func (w *opsRequestTraceCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *opsRequestTraceCaptureWriter) WriteString(value string) (int, error) {
	w.capture([]byte(value))
	return w.ResponseWriter.WriteString(value)
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
