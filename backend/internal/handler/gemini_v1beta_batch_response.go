package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func writeGoogleBatchUpstreamResponse(c *gin.Context, res service.GoogleBatchUpstreamResult) {
	switch typed := res.(type) {
	case *service.UpstreamHTTPResult:
		writeUpstreamResponse(c, typed)
	case *service.UpstreamHTTPStreamResult:
		writeGoogleBatchStreamResponse(c, typed)
	default:
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_empty", "Empty upstream response")
	}
}

func writeGoogleBatchStreamResponse(c *gin.Context, res *service.UpstreamHTTPStreamResult) {
	if res == nil || res.Body == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_empty", "Empty upstream response")
		return
	}
	defer func() { _ = res.Body.Close() }()
	for k, vv := range res.Headers {
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
			continue
		}
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	if res.ContentLength >= 0 {
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
	}
	c.Status(res.StatusCode)
	if _, err := io.Copy(c.Writer, res.Body); err != nil {
		_ = c.Error(err)
	}
}
