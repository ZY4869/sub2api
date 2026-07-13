package handler

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type opsCaptureWriter struct {
	gin.ResponseWriter
	limit int
	buf   bytes.Buffer
}

const opsCaptureWriterLimit = 64 * 1024

var opsCaptureWriterPool = sync.Pool{
	New: func() any {
		return &opsCaptureWriter{limit: opsCaptureWriterLimit}
	},
}

func acquireOpsCaptureWriter(rw gin.ResponseWriter) *opsCaptureWriter {
	w, ok := opsCaptureWriterPool.Get().(*opsCaptureWriter)
	if !ok || w == nil {
		w = &opsCaptureWriter{}
	}
	w.ResponseWriter = rw
	w.limit = opsCaptureWriterLimit
	w.buf.Reset()
	return w
}

func releaseOpsCaptureWriter(w *opsCaptureWriter) {
	if w == nil {
		return
	}
	w.ResponseWriter = nil
	w.limit = opsCaptureWriterLimit
	w.buf.Reset()
	opsCaptureWriterPool.Put(w)
}

func (w *opsCaptureWriter) Header() http.Header {
	if w == nil {
		return http.Header{}
	}
	return opsWriterHeader(w.ResponseWriter)
}

func (w *opsCaptureWriter) Status() int {
	if w == nil {
		return http.StatusOK
	}
	return opsWriterStatus(w.ResponseWriter)
}

func (w *opsCaptureWriter) Size() int {
	if w == nil {
		return 0
	}
	return opsWriterSize(w.ResponseWriter)
}

func (w *opsCaptureWriter) Written() bool {
	return w != nil && opsWriterWritten(w.ResponseWriter)
}

func (w *opsCaptureWriter) WriteHeaderNow() {
	if w != nil {
		opsWriterWriteHeaderNow(w.ResponseWriter)
	}
}

func (w *opsCaptureWriter) WriteHeader(code int) {
	if w != nil {
		opsWriterWriteHeader(w.ResponseWriter, code)
	}
}

func (w *opsCaptureWriter) Write(b []byte) (int, error) {
	if w.Status() >= 400 && w.limit > 0 && w.buf.Len() < w.limit {
		remaining := w.limit - w.buf.Len()
		if len(b) > remaining {
			_, _ = w.buf.Write(b[:remaining])
		} else {
			_, _ = w.buf.Write(b)
		}
	}
	if w == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return opsWriterWrite(w.ResponseWriter, b)
}

func (w *opsCaptureWriter) WriteString(s string) (int, error) {
	if w.Status() >= 400 && w.limit > 0 && w.buf.Len() < w.limit {
		remaining := w.limit - w.buf.Len()
		if len(s) > remaining {
			_, _ = w.buf.WriteString(s[:remaining])
		} else {
			_, _ = w.buf.WriteString(s)
		}
	}
	if w == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return opsWriterWriteString(w.ResponseWriter, s)
}

func (w *opsCaptureWriter) Flush() {
	if w != nil {
		opsWriterFlush(w.ResponseWriter)
	}
}

func (w *opsCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w == nil {
		return nil, nil, errOpsCaptureWriterReleased
	}
	return opsWriterHijack(w.ResponseWriter)
}

func (w *opsCaptureWriter) CloseNotify() <-chan bool {
	if w == nil {
		return opsWriterCloseNotify(nil)
	}
	return opsWriterCloseNotify(w.ResponseWriter)
}

func (w *opsCaptureWriter) Pusher() http.Pusher {
	if w == nil {
		return nil
	}
	return opsWriterPusher(w.ResponseWriter)
}
