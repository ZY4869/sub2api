package middleware

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/gin-gonic/gin"
)

func ServerTiming(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled || c == nil || c.Request == nil {
			c.Next()
			return
		}
		collector := servertiming.New(time.Now())
		ctx := servertiming.WithCollector(c.Request.Context(), collector)
		c.Request = c.Request.WithContext(ctx)
		writer := &serverTimingWriter{
			ResponseWriter: c.Writer,
			ctx:            ctx,
		}
		c.Writer = writer
		c.Next()
		writer.finalize()
	}
}

type serverTimingWriter struct {
	gin.ResponseWriter
	ctx       context.Context
	finalized bool
}

func (w *serverTimingWriter) WriteHeader(code int) {
	w.finalize()
	w.ResponseWriter.WriteHeader(code)
}

func (w *serverTimingWriter) WriteHeaderNow() {
	w.finalize()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *serverTimingWriter) Write(data []byte) (int, error) {
	w.finalize()
	return w.ResponseWriter.Write(data)
}

func (w *serverTimingWriter) WriteString(s string) (int, error) {
	w.finalize()
	return w.ResponseWriter.WriteString(s)
}

func (w *serverTimingWriter) Flush() {
	w.finalize()
	w.ResponseWriter.Flush()
}

func (w *serverTimingWriter) finalize() {
	if w.finalized {
		return
	}
	w.finalized = true
	if value := servertiming.HeaderValue(w.ctx, time.Now()); value != "" {
		w.Header().Set(servertiming.HeaderName, value)
	}
}
