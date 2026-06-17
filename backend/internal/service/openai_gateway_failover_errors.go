package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var openAITransportFailoverResponseBody = []byte(`{"error":{"type":"upstream_error","message":"Upstream request failed"}}`)
var openAINonJSONSuccessFailoverResponseBody = []byte(`{"error":{"type":"upstream_error","message":"Upstream returned non-JSON success response"}}`)

func buildOpenAINonJSONSuccessFailoverBody() []byte {
	return append([]byte(nil), openAINonJSONSuccessFailoverResponseBody...)
}

func newOpenAITransportFailoverError(c *gin.Context, account *Account, err error) *UpstreamFailoverError {
	return newOpenAITransportFailoverErrorWithPassthrough(c, account, err, false)
}

func newOpenAITransportFailoverErrorWithPassthrough(c *gin.Context, account *Account, err error, passthrough bool) *UpstreamFailoverError {
	safeErr := "upstream request failed"
	if err != nil {
		safeErr = sanitizeUpstreamErrorMessage(err.Error())
	}
	setOpsUpstreamError(c, 0, safeErr, "")

	event := OpsUpstreamErrorEvent{
		UpstreamStatusCode: 0,
		Passthrough:        passthrough,
		Kind:               "request_error",
		Message:            safeErr,
	}
	if account != nil {
		event.Platform = RoutingPlatformForAccount(account)
		if event.Platform == "" {
			event.Platform = account.Platform
		}
		event.AccountID = account.ID
		event.AccountName = account.Name
	}
	appendOpsUpstreamError(c, event)

	return &UpstreamFailoverError{
		StatusCode:             http.StatusBadGateway,
		ResponseBody:           openAITransportFailoverResponseBody,
		RetryableOnSameAccount: false,
		TempUnscheduleAccount:  true,
	}
}
