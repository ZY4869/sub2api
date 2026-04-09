package routes

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/localizationaudit"
	"github.com/stretchr/testify/require"
)

var gatewayRawMessagePattern = regexp.MustCompile(`"message":\s*"[^"]+"`)

func TestGatewayLocalizationAuditManifest(t *testing.T) {
	require.Equal(t, []string{
		"internal/server/routes/gateway.go",
		"internal/handler/gemini_v1beta_handler.go",
		"internal/handler/gemini_v1beta_batch_handler.go",
		"internal/handler/gemini_v1beta_batch_response.go",
	}, localizationaudit.GatewayFiles)

	require.Equal(t, []localizationaudit.ExactLiteral{
		{
			File:    "internal/handler/gemini_v1beta_handler.go",
			Sink:    localizationaudit.SinkGeminiDirectGoogleError,
			Literal: "googleError(c, respCode, msg)",
		},
	}, localizationaudit.ExactLiteralAllowlist)
}

func TestGatewayRouteErrorsUseLocalizationHelper(t *testing.T) {
	root := backendRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "internal/server/routes/gateway.go"))
	require.NoError(t, err)

	offenders := make([]string, 0)
	forEachNonCommentLine(string(body), func(lineNo int, trimmed string) {
		if !gatewayRawMessagePattern.MatchString(trimmed) {
			return
		}
		if localizationaudit.IsExactLiteralAllowlisted("internal/server/routes/gateway.go", localizationaudit.SinkGatewayRawMessage, trimmed) {
			return
		}
		offenders = append(offenders, "internal/server/routes/gateway.go:"+strconv.Itoa(lineNo)+":"+trimmed)
	})

	require.Empty(t, offenders)
}

func TestGeminiGatewayHandlersUseLocalizedErrorKeys(t *testing.T) {
	root := backendRoot(t)
	offenders := make([]string, 0)

	for _, file := range localizationaudit.GatewayFiles[1:] {
		body, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err)

		forEachNonCommentLine(string(body), func(lineNo int, trimmed string) {
			if !strings.Contains(trimmed, "googleError(") || strings.HasPrefix(trimmed, "func googleError(") {
				return
			}
			if localizationaudit.IsExactLiteralAllowlisted(file, localizationaudit.SinkGeminiDirectGoogleError, trimmed) {
				return
			}
			offenders = append(offenders, file+":"+strconv.Itoa(lineNo)+":"+trimmed)
		})
	}

	require.Empty(t, offenders)
}

func backendRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "failed to find backend go.mod from %s", dir)
		dir = parent
	}
}

func forEachNonCommentLine(source string, visit func(lineNo int, trimmed string)) {
	inBlockComment := false

	for lineNo, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		for {
			if inBlockComment {
				if end := strings.Index(trimmed, "*/"); end >= 0 {
					trimmed = strings.TrimSpace(trimmed[end+2:])
					inBlockComment = false
				} else {
					trimmed = ""
					break
				}
			}

			start := strings.Index(trimmed, "/*")
			if start < 0 {
				break
			}
			end := strings.Index(trimmed[start+2:], "*/")
			if end < 0 {
				trimmed = strings.TrimSpace(trimmed[:start])
				inBlockComment = true
				break
			}
			trimmed = strings.TrimSpace(trimmed[:start] + trimmed[start+2+end+2:])
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		visit(lineNo+1, trimmed)
	}
}
