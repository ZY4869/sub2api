package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// OpsErrorLoggerMiddleware records error responses (status >= 400) into ops_error_logs.
//
// Notes:
// - It buffers response bodies only when status >= 400 to avoid overhead for successful traffic.
// - Streaming errors after the response has started (SSE) may still need explicit logging.
func OpsErrorLoggerMiddleware(ops *service.OpsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		originalWriter := c.Writer
		w := acquireOpsCaptureWriter(originalWriter)
		defer func() {
			// Restore the original writer before returning so outer middlewares
			// don't observe a pooled wrapper that has been released.
			if c.Writer == w {
				c.Writer = originalWriter
			}
			releaseOpsCaptureWriter(w)
		}()
		c.Writer = w
		c.Next()

		if ops == nil {
			return
		}
		if !ops.IsMonitoringEnabled(c.Request.Context()) {
			return
		}

		status := c.Writer.Status()
		if status < 400 {
			entry := buildRecoveredOpsUpstreamErrorEntry(c, status)
			if entry == nil {
				return
			}
			enqueueOpsErrorLog(ops, entry)
			return
		}

		body := w.buf.Bytes()
		parsed := parseOpsErrorResponse(body)

		// Skip logging if a passthrough rule with skip_monitoring=true matched.
		if shouldSkipOpsPassthrough(c) {
			return
		}

		// Skip logging if the error should be filtered based on settings
		if shouldSkipOpsErrorLog(c.Request.Context(), ops, parsed.Message, string(body), c.Request.URL.Path) {
			return
		}

		entry := buildOpsErrorResponseLogEntry(c, status, body, parsed)
		if entry == nil {
			return
		}
		enqueueOpsErrorLog(ops, entry)
	}
}
