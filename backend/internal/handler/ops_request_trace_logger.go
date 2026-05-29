package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func OpsRequestTraceMiddleware(ops *service.OpsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ops == nil {
			c.Next()
			return
		}

		service.SetOpsTracePayloadPreviewLimit(c, resolveOpsRequestTracePreviewLimit(ops, c.Request.Context()))
		startedAt := time.Now()
		originalWriter := c.Writer
		writer := acquireOpsRequestTraceCaptureWriter(originalWriter)
		c.Writer = writer
		defer func() {
			if c.Writer == writer {
				c.Writer = originalWriter
			}
			releaseOpsRequestTraceCaptureWriter(writer)
		}()

		c.Next()

		if !ops.IsMonitoringEnabled(c.Request.Context()) {
			return
		}

		input := buildOpsRequestTraceInput(ops, c, writer, startedAt)
		if !shouldQueueOpsRequestTrace(input) {
			return
		}
		enqueueOpsRequestTrace(ops, input)
	}
}
