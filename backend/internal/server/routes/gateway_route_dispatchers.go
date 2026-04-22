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

func (d gatewayRouteDispatchers) GatewayV1ModelsList(c *gin.Context) {
	d.handlers.Gateway.GatewayV1ModelsList(c)
}

func (d gatewayRouteDispatchers) GatewayV1ModelsGet(c *gin.Context) {
	d.handlers.Gateway.GatewayV1ModelsGet(c)
}

func (d gatewayRouteDispatchers) GeminiModelsGet(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaGetModel(c)
}

func (d gatewayRouteDispatchers) GatewayV1ModelsAction(c *gin.Context) {
	d.handlers.Gateway.GatewayV1ModelsAction(c)
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

func (d gatewayRouteDispatchers) PublicImagesGeneration(c *gin.Context) {
	d.handlers.DispatchPublicImagesGeneration(c)
}

func (d gatewayRouteDispatchers) PublicImagesEdits(c *gin.Context) {
	d.handlers.DispatchPublicImagesEdits(c)
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

func (d gatewayRouteDispatchers) GeminiEmbeddings(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaEmbeddings(c, "")
}

func (d gatewayRouteDispatchers) GeminiCachedContents(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaCachedContents(c)
}

func (d gatewayRouteDispatchers) GeminiFileSearchStores(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaFileSearchStores(c)
}

func (d gatewayRouteDispatchers) GeminiDocuments(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaDocuments(c)
}

func (d gatewayRouteDispatchers) GeminiOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaOperations(c)
}

func (d gatewayRouteDispatchers) GeminiUploadOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaUploadOperations(c)
}

func (d gatewayRouteDispatchers) GeminiInteractions(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaInteractions(c)
}

func (d gatewayRouteDispatchers) GeminiCorpora(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaCorpora(c)
}

func (d gatewayRouteDispatchers) GeminiCorporaOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaCorporaOperations(c)
}

func (d gatewayRouteDispatchers) GeminiCorporaPermissions(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaCorporaPermissions(c)
}

func (d gatewayRouteDispatchers) GeminiDynamic(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaDynamic(c)
}

func (d gatewayRouteDispatchers) GeminiGeneratedFiles(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaGeneratedFiles(c)
}

func (d gatewayRouteDispatchers) GeminiGeneratedFilesOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaGeneratedFilesOperations(c)
}

func (d gatewayRouteDispatchers) GeminiModelOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaModelOperations(c)
}

func (d gatewayRouteDispatchers) GeminiTunedModels(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaTunedModels(c)
}

func (d gatewayRouteDispatchers) GeminiTunedModelsPermissions(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaTunedModelsPermissions(c)
}

func (d gatewayRouteDispatchers) GeminiTunedModelsOperations(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaTunedModelsOperations(c)
}

func (d gatewayRouteDispatchers) GeminiOpenAICompat(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaOpenAICompat(c)
}

func (d gatewayRouteDispatchers) GeminiLive(c *gin.Context) {
	d.handlers.Gateway.GeminiV1BetaLive(c)
}

func (d gatewayRouteDispatchers) GeminiLiveAuthTokens(c *gin.Context) {
	d.handlers.Gateway.GeminiV1AlphaAuthTokens(c)
}

func (d gatewayRouteDispatchers) GoogleBatchArchiveBatch(c *gin.Context) {
	d.handlers.Gateway.GoogleBatchArchiveBatch(c)
}

func (d gatewayRouteDispatchers) GoogleBatchArchiveFileDownload(c *gin.Context) {
	d.handlers.Gateway.GoogleBatchArchiveFileDownload(c)
}

func (d gatewayRouteDispatchers) VertexModels(c *gin.Context) {
	d.handlers.Gateway.VertexModels(c)
}

func (d gatewayRouteDispatchers) VertexModelsSimplified(c *gin.Context) {
	d.handlers.Gateway.VertexModelsSimplified(c)
}

func (d gatewayRouteDispatchers) VertexBatchPredictionJobs(c *gin.Context) {
	d.handlers.Gateway.VertexBatchPredictionJobs(c)
}

func (d gatewayRouteDispatchers) VertexBatchPredictionJobsSimplified(c *gin.Context) {
	d.handlers.Gateway.VertexBatchPredictionJobsSimplified(c)
}
