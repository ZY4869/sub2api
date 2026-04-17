package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/tidwall/gjson"
)

type baiduDocumentAIClient struct {
	httpUpstream                 HTTPUpstream
	tlsFingerprintProfileService *TLSFingerprintProfileService
}

type baiduDocumentAIAsyncSubmitResult struct {
	ProviderJobID     string
	ProviderBatchID   string
	Status            string
	ProviderRequestID string
	ProviderRawJSON   string
}

type baiduDocumentAIAsyncStatusResult struct {
	Status            string
	ProviderRequestID string
	ProviderRawJSON   string
	JSONResultURL     string
	MarkdownResultURL string
}

type baiduDocumentAIDirectResult struct {
	ProviderRequestID string
	ProviderRawJSON   string
	Envelope          DocumentAIResultEnvelope
}

type baiduDocumentAIProviderError struct {
	err                error
	providerResultJSON string
}

func (e *baiduDocumentAIProviderError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *baiduDocumentAIProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newBaiduDocumentAIClient(httpUpstream HTTPUpstream, tlsFingerprintProfileService *TLSFingerprintProfileService) *baiduDocumentAIClient {
	return &baiduDocumentAIClient{
		httpUpstream:                 httpUpstream,
		tlsFingerprintProfileService: tlsFingerprintProfileService,
	}
}

func (c *baiduDocumentAIClient) submitAsyncJob(ctx context.Context, account *Account, input DocumentAISubmitJobInput) (*baiduDocumentAIAsyncSubmitResult, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream client is not configured")
	}
	if account == nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "document ai account is required")
	}
	baseURL, err := urlvalidator.ValidateHTTPURL(account.GetBaiduDocumentAIAsyncBaseURL(), false, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "invalid baidu document ai async_base_url").WithMetadata(map[string]string{
			"provider_message": err.Error(),
		})
	}
	token := account.GetBaiduDocumentAIAsyncBearerToken()
	if token == "" {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "baidu document ai async token is missing")
	}
	providerModelID := documentAIProviderModelID(input.Model)
	if providerModelID == "" {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "unsupported document ai model")
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/jobs"
	var req *http.Request
	switch input.SourceType {
	case DocumentAISourceTypeFile:
		req, err = c.buildAsyncMultipartRequest(ctx, endpoint, token, providerModelID, input)
	case DocumentAISourceTypeFileURL:
		req, err = c.buildAsyncFileURLRequest(ctx, endpoint, token, providerModelID, input)
	default:
		err = infraerrors.BadRequest("document_ai_invalid_request", "unsupported document ai source_type")
	}
	if err != nil {
		return nil, err
	}

	body, headers, err := c.doJSONRequest(req, account)
	if err != nil {
		return nil, wrapBaiduDocumentAIProviderError(err, body)
	}

	status := normalizeBaiduAsyncJobStatus(firstNonEmptyString(
		gjson.GetBytes(body, "status").String(),
		gjson.GetBytes(body, "state").String(),
		gjson.GetBytes(body, "data.status").String(),
		gjson.GetBytes(body, "data.state").String(),
	))
	result := &baiduDocumentAIAsyncSubmitResult{
		ProviderJobID: firstNonEmptyString(
			gjson.GetBytes(body, "jobId").String(),
			gjson.GetBytes(body, "data.jobId").String(),
			gjson.GetBytes(body, "data.id").String(),
		),
		ProviderBatchID: firstNonEmptyString(
			gjson.GetBytes(body, "batchId").String(),
			gjson.GetBytes(body, "data.batchId").String(),
			trimStringMapValue(input.Options, "batchId"),
		),
		Status:            status,
		ProviderRequestID: readBaiduDocumentAIRequestID(body, headers),
		ProviderRawJSON:   strings.TrimSpace(string(body)),
	}
	if result.ProviderJobID == "" {
		return nil, buildBaiduDocumentAIProviderError(http.StatusBadGateway, body, headers, false)
	}
	if result.Status == "" {
		result.Status = DocumentAIJobStatusPending
	}
	return result, nil
}

func (c *baiduDocumentAIClient) getAsyncJobStatus(ctx context.Context, account *Account, providerJobID string) (*baiduDocumentAIAsyncStatusResult, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream client is not configured")
	}
	if account == nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "document ai account is required")
	}
	baseURL, err := urlvalidator.ValidateHTTPURL(account.GetBaiduDocumentAIAsyncBaseURL(), false, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "invalid baidu document ai async_base_url").WithMetadata(map[string]string{
			"provider_message": err.Error(),
		})
	}
	token := account.GetBaiduDocumentAIAsyncBearerToken()
	if token == "" {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "baidu document ai async token is missing")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/jobs/"+providerJobID, nil)
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to build document ai async status request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	body, headers, err := c.doJSONRequest(req, account)
	if err != nil {
		result := &baiduDocumentAIAsyncStatusResult{
			Status: normalizeBaiduAsyncJobStatus(firstNonEmptyString(
				gjson.GetBytes(body, "status").String(),
				gjson.GetBytes(body, "state").String(),
				gjson.GetBytes(body, "data.status").String(),
				gjson.GetBytes(body, "data.state").String(),
			)),
			ProviderRequestID: readBaiduDocumentAIRequestID(body, headers),
			ProviderRawJSON:   strings.TrimSpace(string(body)),
			JSONResultURL: strings.TrimSpace(firstNonEmptyString(
				gjson.GetBytes(body, "resultUrl.jsonUrl").String(),
				gjson.GetBytes(body, "data.resultUrl.jsonUrl").String(),
			)),
			MarkdownResultURL: strings.TrimSpace(firstNonEmptyString(
				gjson.GetBytes(body, "resultUrl.markdownUrl").String(),
				gjson.GetBytes(body, "data.resultUrl.markdownUrl").String(),
			)),
		}
		if result.Status == "" && result.ProviderRawJSON == "" && result.ProviderRequestID == "" {
			result = nil
		}
		return result, wrapBaiduDocumentAIProviderError(err, body)
	}

	result := &baiduDocumentAIAsyncStatusResult{
		Status: normalizeBaiduAsyncJobStatus(firstNonEmptyString(
			gjson.GetBytes(body, "status").String(),
			gjson.GetBytes(body, "state").String(),
			gjson.GetBytes(body, "data.status").String(),
			gjson.GetBytes(body, "data.state").String(),
		)),
		ProviderRequestID: readBaiduDocumentAIRequestID(body, headers),
		ProviderRawJSON:   strings.TrimSpace(string(body)),
		JSONResultURL: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "resultUrl.jsonUrl").String(),
			gjson.GetBytes(body, "data.resultUrl.jsonUrl").String(),
		)),
		MarkdownResultURL: strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "resultUrl.markdownUrl").String(),
			gjson.GetBytes(body, "data.resultUrl.markdownUrl").String(),
		)),
	}
	if result.Status == "" {
		result.Status = DocumentAIJobStatusPending
	}
	if result.Status == DocumentAIJobStatusFailed {
		return result, buildBaiduDocumentAIProviderError(http.StatusBadGateway, body, headers, true)
	}
	return result, nil
}

func (c *baiduDocumentAIClient) parseDirect(ctx context.Context, account *Account, input DocumentAIParseDirectInput) (*baiduDocumentAIDirectResult, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream client is not configured")
	}
	if account == nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "document ai account is required")
	}
	token := account.GetBaiduDocumentAIDirectToken()
	if token == "" {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "baidu document ai direct token is missing")
	}
	apiURL := account.GetBaiduDocumentAIDirectAPIURL(input.Model)
	if apiURL == "" {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "direct API URL is not configured for this model")
	}
	validatedURL, err := urlvalidator.ValidateHTTPURL(apiURL, false, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "invalid baidu document ai direct_api_url").WithMetadata(map[string]string{
			"provider_message": err.Error(),
		})
	}
	payload := map[string]any{
		"fileType": mapDocumentAIFileType(input.FileType),
	}
	switch input.SourceType {
	case DocumentAISourceTypeFile:
		payload["file"] = base64.StdEncoding.EncodeToString(input.FileBytes)
	case DocumentAISourceTypeFileBase64:
		payload["file"] = strings.TrimSpace(input.FileBase64)
	default:
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "unsupported direct source_type")
	}
	mergeDocumentAIDirectOptions(payload, input.Options)
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai direct payload")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validatedURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to build document ai direct request")
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	body, headers, err := c.doJSONRequest(req, account)
	if err != nil {
		return nil, wrapBaiduDocumentAIProviderError(err, body)
	}
	result := &baiduDocumentAIDirectResult{
		ProviderRequestID: readBaiduDocumentAIRequestID(body, headers),
		ProviderRawJSON:   strings.TrimSpace(string(body)),
		Envelope: DocumentAIResultEnvelope{
			Provider: DocumentAIProviderBaidu,
			Mode:     DocumentAIJobModeDirect,
			Model:    normalizeDocumentAIModelID(input.Model),
			Status:   DocumentAIJobStatusSucceeded,
		},
	}
	normalizeDocumentAIEnvelopeFromJSON(&result.Envelope, body)
	return result, nil
}

func (c *baiduDocumentAIClient) downloadResultText(ctx context.Context, account *Account, rawURL string) (string, error) {
	body, _, err := c.downloadResultURL(ctx, account, rawURL)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *baiduDocumentAIClient) downloadResultJSON(ctx context.Context, account *Account, rawURL string) ([]byte, error) {
	body, _, err := c.downloadResultURL(ctx, account, rawURL)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *baiduDocumentAIClient) downloadResultURL(ctx context.Context, account *Account, rawURL string) ([]byte, http.Header, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream client is not configured")
	}
	validatedURL, err := urlvalidator.ValidateHTTPURL(rawURL, false, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, nil, infraerrors.BadRequest("document_ai_invalid_request", "invalid document ai result URL").WithMetadata(map[string]string{
			"provider_message": err.Error(),
		})
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, validatedURL, nil)
	if err != nil {
		return nil, nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to build document ai result request")
	}
	resp, err := c.doRequest(req, account)
	if err != nil {
		return nil, nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai result fetch failed").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.Header, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai result fetch failed").WithCause(readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, buildBaiduDocumentAIProviderError(resp.StatusCode, body, resp.Header, false)
	}
	return body, resp.Header, nil
}

func (c *baiduDocumentAIClient) buildAsyncMultipartRequest(ctx context.Context, endpoint, token, providerModelID string, input DocumentAISubmitJobInput) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("modelId", providerModelID); err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai multipart request")
	}
	optionalPayload, pageRanges, batchID := buildDocumentAIAsyncOptionPayload(input.Options)
	if pageRanges != "" {
		if err := writer.WriteField("pageRanges", pageRanges); err != nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai multipart request")
		}
	}
	if batchID != "" {
		if err := writer.WriteField("batchId", batchID); err != nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai multipart request")
		}
	}
	if len(optionalPayload) > 0 {
		payloadBytes, err := json.Marshal(optionalPayload)
		if err != nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai optionalPayload")
		}
		if err := writer.WriteField("optionalPayload", string(payloadBytes)); err != nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai multipart request")
		}
	}
	fileWriter, err := writer.CreateFormFile("file", firstNonEmptyString(strings.TrimSpace(input.FileName), "document.bin"))
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai file upload")
	}
	if _, err := fileWriter.Write(input.FileBytes); err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai file upload")
	}
	if err := writer.Close(); err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to finalize document ai multipart request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to build document ai async request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *baiduDocumentAIClient) buildAsyncFileURLRequest(ctx context.Context, endpoint, token, providerModelID string, input DocumentAISubmitJobInput) (*http.Request, error) {
	validatedURL, err := urlvalidator.ValidateHTTPURL(strings.TrimSpace(input.FileURL), false, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, infraerrors.BadRequest("document_ai_invalid_request", "invalid file_url").WithMetadata(map[string]string{
			"provider_message": err.Error(),
		})
	}
	optionalPayload, pageRanges, batchID := buildDocumentAIAsyncOptionPayload(input.Options)
	payload := map[string]any{
		"modelId": providerModelID,
		"fileUrl": validatedURL,
	}
	if len(optionalPayload) > 0 {
		payload["optionalPayload"] = optionalPayload
	}
	if pageRanges != "" {
		payload["pageRanges"] = pageRanges
	}
	if batchID != "" {
		payload["batchId"] = batchID
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to encode document ai file_url payload")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "document_ai_provider_error", "failed to build document ai async request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *baiduDocumentAIClient) doJSONRequest(req *http.Request, account *Account) ([]byte, http.Header, error) {
	resp, err := c.doRequest(req, account)
	if err != nil {
		return nil, nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream request failed").WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.Header, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai upstream request failed").WithCause(readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.Header, buildBaiduDocumentAIProviderError(resp.StatusCode, body, resp.Header, false)
	}
	if providerErrorCode := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "errorCode").String(),
		gjson.GetBytes(body, "code").String(),
	)); providerErrorCode != "" && providerErrorCode != "0" {
		return body, resp.Header, buildBaiduDocumentAIProviderError(resp.StatusCode, body, resp.Header, false)
	}
	return body, resp.Header, nil
}

func wrapBaiduDocumentAIProviderError(err error, body []byte) error {
	if err == nil {
		return nil
	}
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return err
	}
	var wrapped *baiduDocumentAIProviderError
	if errors.As(err, &wrapped) {
		if strings.TrimSpace(wrapped.providerResultJSON) != "" {
			return err
		}
	}
	return &baiduDocumentAIProviderError{
		err:                err,
		providerResultJSON: raw,
	}
}

func documentAIProviderResultJSONFromError(err error) *string {
	var wrapped *baiduDocumentAIProviderError
	if !errors.As(err, &wrapped) {
		return nil
	}
	value := strings.TrimSpace(wrapped.providerResultJSON)
	if value == "" {
		return nil
	}
	return stringPtr(value)
}

func (c *baiduDocumentAIClient) doRequest(req *http.Request, account *Account) (*http.Response, error) {
	if c == nil || c.httpUpstream == nil {
		return nil, fmt.Errorf("document ai upstream client is not configured")
	}
	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return c.httpUpstream.DoWithTLS(
		req,
		proxyURL,
		valueOrZeroInt64(accountIDPtr(account)),
		accountConcurrency(account),
		resolveAccountTLSFingerprintProfile(account, c.tlsFingerprintProfileService),
	)
}

func buildDocumentAIAsyncOptionPayload(options map[string]any) (map[string]any, string, string) {
	if len(options) == 0 {
		return nil, "", ""
	}
	allowed := map[string]struct{}{
		"useDocOrientationClassify": {},
		"useDocUnwarping":           {},
		"useTextlineOrientation":    {},
		"useChartRecognition":       {},
		"useSealRecognition":        {},
		"useTableRecognition":       {},
		"useFormulaRecognition":     {},
		"useRegionDetection":        {},
		"useLayoutDetection":        {},
	}
	optionalPayload := make(map[string]any, len(options))
	pageRanges := ""
	batchID := ""
	for key, value := range options {
		switch strings.TrimSpace(key) {
		case "pageRanges":
			pageRanges = strings.TrimSpace(anyString(value))
		case "batchId":
			batchID = strings.TrimSpace(anyString(value))
		default:
			if _, ok := allowed[strings.TrimSpace(key)]; ok {
				optionalPayload[key] = value
			}
		}
	}
	if len(optionalPayload) == 0 {
		optionalPayload = nil
	}
	return optionalPayload, pageRanges, batchID
}

func mergeDocumentAIDirectOptions(payload map[string]any, options map[string]any) {
	if len(options) == 0 {
		return
	}
	allowed := map[string]struct{}{
		"useDocOrientationClassify": {},
		"useDocUnwarping":           {},
		"useTextlineOrientation":    {},
		"useChartRecognition":       {},
		"useSealRecognition":        {},
		"useTableRecognition":       {},
		"useFormulaRecognition":     {},
		"useRegionDetection":        {},
		"useLayoutDetection":        {},
	}
	for key, value := range options {
		if _, ok := allowed[strings.TrimSpace(key)]; ok {
			payload[key] = value
		}
	}
}

func mapDocumentAIFileType(fileType string) int {
	if trimLower(fileType) == DocumentAIFileTypePDF {
		return 0
	}
	return 1
}

func normalizeBaiduAsyncJobStatus(value string) string {
	switch trimLower(value) {
	case "pending", "queued", "created", "submitted":
		return DocumentAIJobStatusPending
	case "running", "processing", "in_progress":
		return DocumentAIJobStatusRunning
	case "done", "succeeded", "success", "completed":
		return DocumentAIJobStatusSucceeded
	case "failed", "error":
		return DocumentAIJobStatusFailed
	case "canceled", "cancelled":
		return DocumentAIJobStatusCanceled
	default:
		return ""
	}
}

func normalizeDocumentAIEnvelopeFromJSON(envelope *DocumentAIResultEnvelope, body []byte) {
	if envelope == nil {
		return
	}
	if envelope.Provider == "" {
		envelope.Provider = DocumentAIProviderBaidu
	}
	if envelope.Status == "" {
		envelope.Status = DocumentAIJobStatusSucceeded
	}
	if envelope.Text == "" {
		envelope.Text = strings.TrimSpace(firstNonEmptyString(
			gjson.GetBytes(body, "result.markdown").String(),
			gjson.GetBytes(body, "result.markdownText").String(),
			gjson.GetBytes(body, "result.text").String(),
			gjson.GetBytes(body, "result.content").String(),
			gjson.GetBytes(body, "data.result.text").String(),
		))
	}
	if envelope.PageCount == 0 {
		envelope.PageCount = firstPositiveInt(
			len(gjson.GetBytes(body, "result.layoutParsingResults").Array()),
			len(gjson.GetBytes(body, "result.ocrResults").Array()),
			len(gjson.GetBytes(body, "data.result.layoutParsingResults").Array()),
		)
	}
	if envelope.PageCount == 0 && strings.TrimSpace(envelope.Text) != "" {
		envelope.PageCount = 1
	}
	if envelope.TableCount == 0 {
		envelope.TableCount = countTableHints(gjson.ParseBytes(body).Value())
	}
	if !envelope.HasLayout {
		envelope.HasLayout = len(gjson.GetBytes(body, "result.layoutParsingResults").Array()) > 0 ||
			len(gjson.GetBytes(body, "data.result.layoutParsingResults").Array()) > 0 ||
			countLayoutHints(gjson.ParseBytes(body).Value()) > 0
	}
}

func buildBaiduDocumentAIProviderError(statusCode int, body []byte, headers http.Header, jobFailed bool) error {
	providerCode := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "errorCode").String(),
		gjson.GetBytes(body, "code").String(),
		gjson.GetBytes(body, "error.code").String(),
	))
	providerMessage := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "errorMsg").String(),
		gjson.GetBytes(body, "msg").String(),
		gjson.GetBytes(body, "message").String(),
		gjson.GetBytes(body, "error.message").String(),
		string(body),
	))
	providerRequestID := readBaiduDocumentAIRequestID(body, headers)
	reason := "document_ai_provider_error"
	httpCode := http.StatusBadGateway
	switch {
	case jobFailed:
		reason = "document_ai_job_failed"
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		reason = "document_ai_auth_error"
		httpCode = http.StatusBadGateway
	case statusCode == http.StatusTooManyRequests:
		reason = "document_ai_rate_limited"
		httpCode = http.StatusTooManyRequests
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		reason = "document_ai_invalid_request"
		httpCode = http.StatusBadRequest
	}
	if providerCode != "" {
		switch trimLower(providerCode) {
		case "401", "403", "unauthorized", "forbidden":
			reason = "document_ai_auth_error"
			httpCode = http.StatusBadGateway
		case "429", "12002":
			reason = "document_ai_rate_limited"
			httpCode = http.StatusTooManyRequests
		case "400", "422", "1001", "1002", "1003", "11002":
			reason = "document_ai_invalid_request"
			httpCode = http.StatusBadRequest
		}
	}
	if strings.TrimSpace(providerMessage) == "" {
		providerMessage = http.StatusText(statusCode)
	}
	message := "document ai provider request failed"
	switch reason {
	case "document_ai_auth_error":
		message = "document ai authentication failed"
	case "document_ai_invalid_request":
		message = "document ai request is invalid"
	case "document_ai_rate_limited":
		message = "document ai provider is rate limited"
	case "document_ai_job_failed":
		message = "document ai provider job failed"
	}
	metadata := map[string]string{
		"provider_code":    providerCode,
		"provider_message": providerMessage,
	}
	if providerRequestID != "" {
		metadata["provider_request_id"] = providerRequestID
	}
	return infraerrors.New(httpCode, reason, message).WithMetadata(metadata)
}

func readBaiduDocumentAIRequestID(body []byte, headers http.Header) string {
	return strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "logId").String(),
		gjson.GetBytes(body, "traceId").String(),
		gjson.GetBytes(body, "requestId").String(),
		gjson.GetBytes(body, "data.requestId").String(),
		headers.Get("X-Request-Id"),
		headers.Get("x-request-id"),
	))
}

func countTableHints(value any) int {
	switch typed := value.(type) {
	case map[string]any:
		total := 0
		for key, item := range typed {
			lowerKey := trimLower(key)
			if strings.Contains(lowerKey, "table") {
				switch cast := item.(type) {
				case []any:
					total += len(cast)
				default:
					total++
				}
			}
			total += countTableHints(item)
		}
		return total
	case []any:
		total := 0
		for _, item := range typed {
			total += countTableHints(item)
		}
		return total
	default:
		return 0
	}
}

func countLayoutHints(value any) int {
	switch typed := value.(type) {
	case map[string]any:
		total := 0
		for key, item := range typed {
			lowerKey := trimLower(key)
			if strings.Contains(lowerKey, "layout") || strings.Contains(lowerKey, "bbox") {
				switch cast := item.(type) {
				case []any:
					total += len(cast)
				default:
					total++
				}
			}
			total += countLayoutHints(item)
		}
		return total
	case []any:
		total := 0
		for _, item := range typed {
			total += countLayoutHints(item)
		}
		return total
	default:
		return 0
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func accountIDPtr(account *Account) *int64 {
	if account == nil {
		return nil
	}
	return &account.ID
}

func accountConcurrency(account *Account) int {
	if account == nil {
		return 0
	}
	return account.Concurrency
}

func valueOrZeroInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func trimStringMapValue(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(anyString(values[key]))
}
