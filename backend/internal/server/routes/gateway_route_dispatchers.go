package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type gatewayRouteDispatchers struct {
	handlers *handler.Handlers
}

func newGatewayRouteDispatchers(h *handler.Handlers) gatewayRouteDispatchers {
	return gatewayRouteDispatchers{handlers: h}
}

func (d gatewayRouteDispatchers) AnthropicMessages(c *gin.Context) {
	dispatchMessagesRoute(c, d.handlers.Gateway.Messages, d.handlers.OpenAIGateway.Messages)
}

func (d gatewayRouteDispatchers) AnthropicCountTokens(c *gin.Context) {
	dispatchCountTokensRoute(c, d.handlers.Gateway.CountTokens)
}

func (d gatewayRouteDispatchers) OpenAIResponses(c *gin.Context) {
	dispatchOpenAIRoute(c, service.EndpointResponses, service.ProtocolCapabilityActionDefault, d.handlers.OpenAIGateway.Responses, d.handlers.GrokGateway.Responses)
}

func (d gatewayRouteDispatchers) OpenAIResponsesWebSocket(c *gin.Context) {
	dispatchOpenAIRoute(c, service.EndpointResponses, service.ProtocolCapabilityActionWebSocket, d.handlers.OpenAIGateway.ResponsesWebSocket, nil)
}

func (d gatewayRouteDispatchers) OpenAIChatCompletions(c *gin.Context) {
	dispatchOpenAIRoute(c, service.EndpointChatCompletions, service.ProtocolCapabilityActionDefault, d.handlers.OpenAIGateway.ChatCompletions, d.handlers.GrokGateway.ChatCompletions)
}

func (d gatewayRouteDispatchers) GrokImagesGeneration(c *gin.Context) {
	dispatchGrokOnlyRoute(c, service.EndpointImagesGen, d.handlers.GrokGateway.ImagesGeneration)
}

func (d gatewayRouteDispatchers) GrokImagesEdits(c *gin.Context) {
	dispatchGrokOnlyRoute(c, service.EndpointImagesEdits, d.handlers.GrokGateway.ImagesEdits)
}

func (d gatewayRouteDispatchers) GrokVideosGeneration(c *gin.Context) {
	dispatchGrokOnlyRoute(c, service.EndpointVideosCreate, d.handlers.GrokGateway.VideosGeneration)
}

func (d gatewayRouteDispatchers) GrokVideosStatus(c *gin.Context) {
	dispatchGrokOnlyRoute(c, service.EndpointVideosStatus, d.handlers.GrokGateway.VideoStatus)
}

func (d gatewayRouteDispatchers) GeminiModels(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaModels(c)
}

func (d gatewayRouteDispatchers) GeminiFiles(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaFiles(c)
}

func (d gatewayRouteDispatchers) GeminiFilesUpload(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaFileUpload(c)
}

func (d gatewayRouteDispatchers) GeminiFilesDownload(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaFileDownload(c)
}

func (d gatewayRouteDispatchers) GeminiBatches(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaBatches(c)
}

func (d gatewayRouteDispatchers) GoogleBatchArchiveBatch(c *gin.Context) {
	d.handlers.Gateway.GoogleBatchArchiveBatch(c)
}

func (d gatewayRouteDispatchers) GoogleBatchArchiveFileDownload(c *gin.Context) {
	d.handlers.Gateway.GoogleBatchArchiveFileDownload(c)
}

func (d gatewayRouteDispatchers) VertexModels(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaModels(c)
}

func (d gatewayRouteDispatchers) VertexBatchPredictionJobs(c *gin.Context) {
	d.handlers.Gateway.VertexBatchPredictionJobs(c)
}
