package handler

import (
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Embeddings handles OpenAI-compatible embeddings requests.
func (h *OpenAIGatewayHandler) Embeddings(c *gin.Context) {
	requestStart := time.Now()
	streamStarted := false

	prepared, ok := h.prepareEmbeddingsRequest(c)
	if !ok {
		return
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, prepared.subject.UserID, prepared.subject.Concurrency, false, &streamStarted, prepared.reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	h.runEmbeddingsRoutingLoop(c, prepared, subscription, &streamStarted)
}
