package admin

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/localizationaudit"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/stretchr/testify/require"
)

var (
	adminRawResponseMessagePattern = regexp.MustCompile(`response\.(BadRequest|Error|InternalError|NotFound)\(c,\s*"|gin\.H\{"message":\s*"`)
	serviceInfraReasonPattern      = regexp.MustCompile(`infraerrors\.(BadRequest|Unauthorized|Forbidden|Conflict|NotFound|ServiceUnavailable|InternalServer|TooManyRequests)\("([^"]+)"`)
)

func TestAccountHandlerLocalizationAuditManifest(t *testing.T) {
	require.Equal(t, []string{
		"internal/handler/admin/account_handler.go",
		"internal/handler/admin/account_handler_batch_ops.go",
		"internal/handler/admin/account_handler_blacklist.go",
		"internal/handler/admin/account_handler_crud.go",
		"internal/handler/admin/account_handler_model_import.go",
		"internal/handler/admin/account_handler_runtime.go",
		"internal/handler/admin/account_handler_runtime_actions.go",
	}, localizationaudit.AdminHandlerFiles)

	require.Equal(t, []string{
		"internal/service/account_test_service.go",
		"internal/service/account_test_models.go",
		"internal/service/account_test_real_forward.go",
		"internal/service/account_test_runtime_meta.go",
		"internal/service/account_model_import_service.go",
		"internal/service/account_model_import_probe.go",
		"internal/service/account_model_import_error_metadata.go",
		"internal/service/account_blacklist_advice.go",
	}, localizationaudit.AdminServiceFiles)
}

func TestTouchedAccountHandlersUseLocalizationHelpers(t *testing.T) {
	root := backendRoot(t)
	offenders := make([]string, 0)

	for _, file := range localizationaudit.AdminHandlerFiles {
		body, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err)

		forEachNonCommentLine(string(body), func(lineNo int, trimmed string) {
			if !adminRawResponseMessagePattern.MatchString(trimmed) {
				return
			}
			if localizationaudit.IsExactLiteralAllowlisted(file, localizationaudit.SinkAdminRawResponse, trimmed) {
				return
			}
			offenders = append(offenders, file+":"+strconv.Itoa(lineNo)+":"+trimmed)
		})
	}

	require.Empty(t, offenders)
}

func TestTouchedAccountServicesExposeLocalizedReasons(t *testing.T) {
	root := backendRoot(t)
	offenders := make([]string, 0)

	for _, file := range localizationaudit.AdminServiceFiles {
		body, err := os.ReadFile(filepath.Join(root, file))
		require.NoError(t, err)

		for lineNo, line := range strings.Split(string(body), "\n") {
			matches := serviceInfraReasonPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				reason := strings.TrimSpace(match[2])
				if reason == "" {
					continue
				}
				if response.HasLocalizedReasonMessage(reason) {
					continue
				}
				if localizationaudit.IsExactLiteralAllowlisted(file, localizationaudit.SinkServiceInfraReason, reason) {
					continue
				}
				offenders = append(offenders, file+":"+strconv.Itoa(lineNo+1)+":"+reason)
			}
		}
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
