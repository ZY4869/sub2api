package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

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
