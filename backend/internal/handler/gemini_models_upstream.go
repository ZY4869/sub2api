package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type geminiUpstreamModelsPayload struct {
	Models        []map[string]any `json:"models"`
	NextPageToken string           `json:"nextPageToken,omitempty"`
}

func (h *GatewayHandler) fetchGeminiUpstreamModelsList(
	c *gin.Context,
	apiKey *service.APIKey,
	effectivePlatform string,
	visibleEntries []service.APIKeyPublicModelEntry,
	pageSize int,
	rawPageToken string,
) (*service.UpstreamHTTPResult, bool) {
	if h == nil || h.gatewayService == nil || h.geminiNativeService == nil || c == nil {
		return nil, false
	}
	account, listTokenSafe, err := h.gatewayService.ResolveGeminiPublicModelMetadataAccount(c.Request.Context(), apiKey, effectivePlatform, "")
	if err != nil || account == nil || !listTokenSafe {
		return nil, false
	}

	visibleNames := make(map[string]struct{}, len(visibleEntries))
	for _, entry := range visibleEntries {
		name := strings.TrimSpace(entry.PublicID)
		if name == "" {
			continue
		}
		visibleNames["models/"+name] = struct{}{}
	}
	if len(visibleNames) == 0 {
		return nil, false
	}

	currentToken := strings.TrimSpace(rawPageToken)
	collected := make([]map[string]any, 0, pageSize)
	nextPageToken := ""

	for attempts := 0; attempts < 20 && len(collected) < pageSize; attempts++ {
		remaining := pageSize - len(collected)
		if remaining <= 0 {
			break
		}
		res, ok := h.fetchGeminiUpstreamModelsPage(c, account, remaining, currentToken)
		if !ok {
			return nil, false
		}
		payload := geminiUpstreamModelsPayload{}
		if err := json.Unmarshal(res.Body, &payload); err != nil {
			return nil, false
		}
		for _, model := range payload.Models {
			name, _ := model["name"].(string)
			if _, ok := visibleNames[strings.TrimSpace(name)]; !ok {
				continue
			}
			collected = append(collected, model)
			if len(collected) >= pageSize {
				break
			}
		}
		nextPageToken = strings.TrimSpace(payload.NextPageToken)
		if nextPageToken == "" {
			break
		}
		currentToken = nextPageToken
	}

	body, err := json.Marshal(geminiUpstreamModelsPayload{
		Models:        collected,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return nil, false
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return &service.UpstreamHTTPResult{StatusCode: http.StatusOK, Headers: headers, Body: body}, true
}

func (h *GatewayHandler) fetchGeminiUpstreamModelsPage(
	c *gin.Context,
	account *service.Account,
	pageSize int,
	pageToken string,
) (*service.UpstreamHTTPResult, bool) {
	if h == nil || h.geminiNativeService == nil || c == nil || account == nil {
		return nil, false
	}
	values := url.Values{}
	if pageSize > 0 {
		values.Set("pageSize", strconv.Itoa(pageSize))
	}
	if strings.TrimSpace(pageToken) != "" {
		values.Set("pageToken", strings.TrimSpace(pageToken))
	}
	path := "/v1beta/models"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	res, err := h.geminiNativeService.ForwardAIStudioGET(c.Request.Context(), account, path)
	if err != nil || res == nil || res.StatusCode != http.StatusOK {
		return nil, false
	}
	return res, true
}

func (h *GatewayHandler) fetchGeminiUpstreamModelDetail(
	c *gin.Context,
	apiKey *service.APIKey,
	effectivePlatform string,
	entry service.APIKeyPublicModelEntry,
) (*service.UpstreamHTTPResult, bool) {
	if h == nil || h.gatewayService == nil || h.geminiNativeService == nil || c == nil {
		return nil, false
	}
	modelID := strings.TrimSpace(firstNonEmptyString(entry.SourceID, entry.PublicID))
	if modelID == "" {
		return nil, false
	}
	account, _, err := h.gatewayService.ResolveGeminiPublicModelMetadataAccount(c.Request.Context(), apiKey, effectivePlatform, entry.PublicID)
	if err != nil || account == nil {
		return nil, false
	}
	path := "/v1beta/models/" + url.PathEscape(modelID)
	res, err := h.geminiNativeService.ForwardAIStudioGET(c.Request.Context(), account, path)
	if err != nil || res == nil || res.StatusCode != http.StatusOK {
		return nil, false
	}
	return res, true
}
