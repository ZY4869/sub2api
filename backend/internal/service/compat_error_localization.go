package service

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func writeLocalizedCompatError(c *gin.Context, writeError compatErrorWriter, errType string, err error) bool {
	if writeError == nil {
		return false
	}
	compatErr, ok := apicompat.AsCompatError(err)
	if !ok || compatErr == nil {
		return false
	}
	message, ok := response.LocalizedCompatErrorMessage(c, err)
	if !ok {
		return false
	}
	statusCode := compatErr.StatusCode
	if statusCode <= 0 {
		statusCode = 400
	}
	writeError(c, statusCode, errType, message, compatErr.Reason)
	return true
}
