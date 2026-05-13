package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type vertexServiceAccountRoundTripFunc func(*http.Request) (*http.Response, error)

func (f vertexServiceAccountRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type vertexServiceAccountProxyRepoStub struct {
	getByIDFunc func(ctx context.Context, id int64) (*Proxy, error)
}

func (s *vertexServiceAccountProxyRepoStub) Create(ctx context.Context, proxy *Proxy) error {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("proxy not found")
}

func (s *vertexServiceAccountProxyRepoStub) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) Update(ctx context.Context, proxy *Proxy) error {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) Delete(ctx context.Context, id int64) error {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	panic("not implemented")
}

func (s *vertexServiceAccountProxyRepoStub) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	panic("not implemented")
}

func TestBuildVertexServiceAccountAssertion_AudIsFixedTokenEndpoint(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	creds := &vertexServiceAccountCredentials{
		Type:        "service_account",
		ClientEmail: "svc@example.com",
		PrivateKey:  string(privateKeyPEM),
		// Malicious / irrelevant: must not affect aud.
		TokenURI: "http://169.254.169.254/latest/meta-data",
	}

	now := time.Unix(1_700_000_000, 0)
	assertion, err := buildVertexServiceAccountAssertion(creds, now)
	require.NoError(t, err)
	require.NotEmpty(t, assertion)

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithoutClaimsValidation(),
	)
	parsed, err := parser.Parse(assertion, func(token *jwt.Token) (any, error) {
		return &key.PublicKey, nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, vertexServiceAccountTokenURL, claims["aud"])
}

func TestGeminiTokenProvider_ExchangeVertexServiceAccountToken_UsesAccountProxy(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	creds := &vertexServiceAccountCredentials{
		Type:        "service_account",
		ClientEmail: "svc@example.com",
		PrivateKey:  string(privateKeyPEM),
		TokenURI:    vertexServiceAccountTokenURL,
	}

	proxyID := int64(42)
	account := &Account{ProxyID: &proxyID}
	proxyRepo := &vertexServiceAccountProxyRepoStub{
		getByIDFunc: func(ctx context.Context, id int64) (*Proxy, error) {
			require.Equal(t, proxyID, id)
			return &Proxy{
				Protocol: "http",
				Host:     "proxy.example.com",
				Port:     8080,
				Status:   StatusActive,
			}, nil
		},
	}
	provider := &GeminiTokenProvider{
		geminiOAuthService: &GeminiOAuthService{proxyRepo: proxyRepo},
	}

	originalGetClient := getVertexServiceAccountHTTPClient
	defer func() {
		getVertexServiceAccountHTTPClient = originalGetClient
	}()

	var capturedProxyURL string
	getVertexServiceAccountHTTPClient = func(opts httpclient.Options) (*http.Client, error) {
		capturedProxyURL = opts.ProxyURL
		return &http.Client{
			Transport: vertexServiceAccountRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, vertexServiceAccountTokenURL, req.URL.String())
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"vertex-token","expires_in":3600}`)),
				}, nil
			}),
		}, nil
	}

	token, ttl, err := provider.exchangeVertexServiceAccountToken(context.Background(), account, creds)
	require.NoError(t, err)
	require.Equal(t, "http://proxy.example.com:8080", capturedProxyURL)
	require.Equal(t, "vertex-token", token)
	require.Positive(t, ttl)
}
