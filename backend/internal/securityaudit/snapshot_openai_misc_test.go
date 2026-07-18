package securityaudit

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExtractPromptSnapshotOpenAIMiscProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		body     string
		want     string
	}{
		{name: "completions", protocol: ProtocolOpenAICompletions, body: `{"prompt":["first","second"]}`, want: "second\n\nfirst"},
		{name: "embeddings", protocol: ProtocolOpenAIEmbeddings, body: `{"input":["embed one","embed two"]}`, want: "embed two\n\nembed one"},
		{name: "alpha search", protocol: ProtocolOpenAIAlphaSearch, body: `{"query":"search this"}`, want: "search this"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body)})

			require.NoError(t, err)
			require.Equal(t, tt.want, snapshot.FullPrompt)
			require.NotEmpty(t, snapshot.PromptHash)
		})
	}
}

func TestExtractPromptSnapshotOpenAIMiscSkipsMediaPayloads(t *testing.T) {
	_, err := ExtractPromptSnapshot(Request{
		Protocol: ProtocolOpenAIEmbeddings,
		Body:     []byte(`{"input":["https://example.com/image.png","data:image/png;base64,aaaa"]}`),
	})

	require.ErrorIs(t, err, ErrNoPromptText)
}

func TestGatewayMiddlewareWhenSkipsAuditWhenPredicateRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newTestPromptService(t, enabledConfig(true, false), &fakeScanner{result: blockResult()}, &fakePromptRepo{})
	router := gin.New()
	router.POST("/test", GatewayMiddlewareWhen(service, ProtocolOpenAIAlphaSearch, func(*gin.Context) bool { return false }), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{"query":"blocked"}`)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
