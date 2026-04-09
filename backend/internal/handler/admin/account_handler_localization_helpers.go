package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func mixedChannelWarningMessage(c *gin.Context, err *service.MixedChannelError) string {
	if err == nil {
		return ""
	}
	return response.LocalizedMessage(
		c,
		"admin.account.mixed_channel_warning",
		err.Error(),
		err.GroupName,
		err.CurrentPlatform,
		err.OtherPlatform,
	)
}
