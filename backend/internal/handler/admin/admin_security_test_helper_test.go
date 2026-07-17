package admin

import "github.com/gin-gonic/gin"

func allowStepUpForHandlerTests() *AdminSecurityHelper {
	return &AdminSecurityHelper{
		stepUpVerifier: func(c *gin.Context, scope string) bool {
			return true
		},
	}
}
