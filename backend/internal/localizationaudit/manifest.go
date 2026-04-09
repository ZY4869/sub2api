package localizationaudit

import "strings"

type Sink string

const (
	SinkGatewayRawMessage       Sink = "gateway_raw_message"
	SinkGeminiDirectGoogleError Sink = "gemini_direct_google_error"
	SinkAdminRawResponse        Sink = "admin_raw_response"
	SinkServiceInfraReason      Sink = "service_infra_reason"
)

type ExactLiteral struct {
	File    string
	Sink    Sink
	Literal string
}

var GatewayFiles = []string{
	"internal/server/routes/gateway.go",
	"internal/handler/gemini_v1beta_handler.go",
	"internal/handler/gemini_v1beta_batch_handler.go",
	"internal/handler/gemini_v1beta_batch_response.go",
}

var AdminHandlerFiles = []string{
	"internal/handler/admin/account_handler.go",
	"internal/handler/admin/account_handler_batch_ops.go",
	"internal/handler/admin/account_handler_blacklist.go",
	"internal/handler/admin/account_handler_crud.go",
	"internal/handler/admin/account_handler_model_import.go",
	"internal/handler/admin/account_handler_runtime.go",
	"internal/handler/admin/account_handler_runtime_actions.go",
}

var AdminServiceFiles = []string{
	"internal/service/account_test_service.go",
	"internal/service/account_test_models.go",
	"internal/service/account_test_real_forward.go",
	"internal/service/account_test_runtime_meta.go",
	"internal/service/account_model_import_service.go",
	"internal/service/account_model_import_probe.go",
	"internal/service/account_model_import_error_metadata.go",
	"internal/service/account_blacklist_advice.go",
}

var ExactLiteralAllowlist = []ExactLiteral{
	{
		File:    "internal/handler/gemini_v1beta_handler.go",
		Sink:    SinkGeminiDirectGoogleError,
		Literal: "googleError(c, respCode, msg)",
	},
}

func IsExactLiteralAllowlisted(file string, sink Sink, literal string) bool {
	file = strings.TrimSpace(file)
	literal = strings.TrimSpace(literal)
	for _, item := range ExactLiteralAllowlist {
		if item.File == file && item.Sink == sink && item.Literal == literal {
			return true
		}
	}
	return false
}
