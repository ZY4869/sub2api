package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// Recovery converts panics into the project's standard JSON error envelope.
//
// It preserves Gin's broken-pipe handling by not attempting to write a response
// when the client connection is already gone.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			recoveredErr, _ := recovered.(error)
			panicSummary := strings.TrimSpace(fmt.Sprint(recovered))
			requestID := ""
			if c != nil && c.Request != nil {
				if value, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
					requestID = strings.TrimSpace(value)
				}
			}

			fmt.Fprintf(
				gin.DefaultErrorWriter,
				"[Recovery] request_id=%s method=%s path=%s client_ip=%s panic=%s\n%s",
				requestID,
				requestMethod(c),
				requestPath(c),
				ip.GetTrustedClientIP(c),
				panicSummary,
				string(debug.Stack()),
			)

			if isBrokenPipe(recoveredErr) {
				if recoveredErr != nil {
					_ = c.Error(recoveredErr)
				}
				c.Abort()
				return
			}

			if c.Writer.Written() {
				c.Abort()
				return
			}

			response.ErrorWithDetails(
				c,
				http.StatusInternalServerError,
				infraerrors.UnknownMessage,
				infraerrors.UnknownReason,
				nil,
			)
			c.Abort()
		}()

		c.Next()
	}
}

func requestMethod(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.Method
}

func requestPath(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return c.Request.URL.Path
}

func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}

	var syscallErr *os.SyscallError
	if !errors.As(opErr.Err, &syscallErr) {
		return false
	}

	msg := strings.ToLower(syscallErr.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
