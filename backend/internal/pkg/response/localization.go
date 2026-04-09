package response

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/gin-gonic/gin"
)

type localizedMessageMap map[string]string

var localizedReasonMessages = map[string]localizedMessageMap{
	"TEST_SOURCE_PROTOCOL_INVALID": {
		"en": "selected source_protocol is not accepted by this account",
		"zh": "当前账号不接受所选 source_protocol，请调整协议后重试。",
	},
	"TEST_TARGET_PROVIDER_INVALID": {
		"en": "selected target_provider is not available for this account",
		"zh": "当前账号没有可用的目标厂商，请重新选择后重试。",
	},
	"TEST_TARGET_PROVIDER_INCOMPATIBLE": {
		"en": "selected target_provider is not compatible with source_protocol",
		"zh": "所选目标厂商与当前 source_protocol 不兼容，请调整后重试。",
	},
	"TEST_TARGET_PROVIDER_REQUIRED": {
		"en": "mixed protocol gateway test requires target_provider or source_protocol",
		"zh": "Mixed 协议网关测试需要明确选择测试厂商或 source_protocol。",
	},
	"TEST_TARGET_MODEL_INVALID": {
		"en": "selected target_model_id is not available for this account",
		"zh": "当前账号没有可用的目标模型，请重新选择后重试。",
	},
	"TEST_TARGET_MODEL_REQUIRED": {
		"en": "selected target_provider does not have a default test model",
		"zh": "所选测试厂商没有可用的默认测试模型，请手动指定模型后重试。",
	},
	"TEST_PROBE_RESOLUTION_FAILED": {
		"en": "mixed protocol gateway test could not resolve a unique protocol",
		"zh": "Mixed 协议网关测试无法解析出唯一协议，请补充厂商、模型或 source_protocol。",
	},
	"ACCOUNT_PROTOCOL_GATEWAY_APIKEY_REQUIRED": {
		"en": "protocol_gateway accounts only support apikey type",
		"zh": "protocol_gateway 账号仅支持 apikey 类型。",
	},
	"ACCOUNT_PROTOCOL_GATEWAY_PROTOCOL_REQUIRED": {
		"en": "protocol_gateway accounts require gateway_protocol",
		"zh": "protocol_gateway 账号必须指定 gateway_protocol。",
	},
	"ACCOUNT_REQUIRED": {
		"en": "account is required",
		"zh": "账户不能为空。",
	},
	"ACCOUNT_INACTIVE": {
		"en": "account must be active before continuing",
		"zh": "账户必须处于启用状态后才能继续。",
	},
	"ACCOUNT_PLATFORM_UNSUPPORTED": {
		"en": "current account platform does not support this operation",
		"zh": "当前账户平台不支持此操作。",
	},
	"ACCOUNT_CREDENTIAL_REQUIRED": {
		"en": "required account credentials are missing",
		"zh": "缺少必要的账户凭证。",
	},
	"ACCOUNT_TYPE_UNSUPPORTED": {
		"en": "current account type does not support this operation",
		"zh": "当前账户类型不支持此操作。",
	},
	"MODEL_CATALOG_SERVICE_UNAVAILABLE": {
		"en": "model catalog service is unavailable",
		"zh": "模型目录服务当前不可用。",
	},
	"MODEL_IMPORT_ANTIGRAVITY_CLIENT_INIT_FAILED": {
		"en": "failed to initialize the Antigravity model probe",
		"zh": "初始化 Antigravity 模型探测失败。",
	},
	"MODEL_IMPORT_EMPTY": {
		"en": "no models were detected for this account",
		"zh": "当前账户未探测到可导入模型。",
	},
	"MODEL_IMPORT_GEMINI_SERVICE_UNAVAILABLE": {
		"en": "gemini model import service is not configured",
		"zh": "Gemini 模型导入服务尚未配置。",
	},
	"MODEL_IMPORT_GEMINI_TOKEN_PROVIDER_UNAVAILABLE": {
		"en": "gemini token provider is not configured",
		"zh": "Gemini Token 提供器尚未配置。",
	},
	"MODEL_IMPORT_HTTP_UPSTREAM_UNAVAILABLE": {
		"en": "model import upstream client is not configured",
		"zh": "模型导入上游客户端尚未配置。",
	},
	"MODEL_IMPORT_INVALID_RESPONSE": {
		"en": "upstream returned an invalid model list response",
		"zh": "上游返回的模型列表响应无效。",
	},
	"MODEL_IMPORT_OPENAI_TOKEN_PROVIDER_UNAVAILABLE": {
		"en": "openai token provider is not configured",
		"zh": "OpenAI Token 提供器尚未配置。",
	},
	"MODEL_IMPORT_PROBE_RESULT_MISSING": {
		"en": "model import probe result is missing",
		"zh": "模型导入探测结果缺失。",
	},
	"MODEL_IMPORT_PROXY_RESOLVE_FAILED": {
		"en": "failed to resolve the account proxy for model import",
		"zh": "解析模型导入代理失败。",
	},
	"MODEL_IMPORT_REQUEST_BUILD_FAILED": {
		"en": "failed to build the upstream model probe request",
		"zh": "构建上游模型探测请求失败。",
	},
	"MODEL_IMPORT_SERVICE_UNAVAILABLE": {
		"en": "account model import service is unavailable",
		"zh": "账户模型导入服务当前不可用。",
	},
	"MODEL_IMPORT_UPSTREAM_EMPTY_RESPONSE": {
		"en": "upstream returned an empty response while listing models",
		"zh": "上游返回了空的模型列表响应。",
	},
	"MODEL_IMPORT_UPSTREAM_FAILED": {
		"en": "upstream model import request failed",
		"zh": "上游模型导入请求失败。",
	},
	"MODEL_IMPORT_UPSTREAM_FORBIDDEN": {
		"en": "upstream rejected the model import request",
		"zh": "上游拒绝了模型导入请求。",
	},
	"MODEL_IMPORT_UPSTREAM_RATE_LIMITED": {
		"en": "upstream rate limit was reached during model import",
		"zh": "模型导入时触发了上游限流。",
	},
	"MODEL_IMPORT_UPSTREAM_READ_FAILED": {
		"en": "failed to read the upstream model list response",
		"zh": "读取上游模型列表响应失败。",
	},
	"MODEL_IMPORT_UPSTREAM_REQUEST_FAILED": {
		"en": "failed to request the upstream model list",
		"zh": "请求上游模型列表失败。",
	},
	"MODEL_IMPORT_UPSTREAM_SERVER_ERROR": {
		"en": "upstream model import service is temporarily unavailable",
		"zh": "上游模型导入服务暂时不可用。",
	},
	"MODEL_IMPORT_UPSTREAM_UNAUTHORIZED": {
		"en": "upstream credentials for model import are invalid",
		"zh": "模型导入使用的上游凭证无效。",
	},
	"MODEL_IMPORT_VERTEX_CATALOG_UNAVAILABLE": {
		"en": "vertex catalog service is not configured",
		"zh": "Vertex 模型目录服务尚未配置。",
	},
}

var localizedMessageKeys = map[string]localizedMessageMap{
	"gateway.grok.messages_unsupported": {
		"en": "Grok groups do not support /v1/messages endpoints",
		"zh": "Grok 分组不支持 /v1/messages 端点。",
	},
	"gateway.grok.alias_reserved": {
		"en": "%s is reserved for Grok groups only",
		"zh": "%s 仅供 Grok 分组使用。",
	},
	"gateway.count_tokens.unsupported_platform": {
		"en": "Token counting is not supported for this platform",
		"zh": "当前平台不支持 token 计数。",
	},
	"gateway.public_endpoint.unsupported_platform": {
		"en": "%s is not supported for this platform",
		"zh": "当前平台不支持 %s。",
	},
	"gateway.public_endpoint.unsupported_action": {
		"en": "%s does not support this action on the current route",
		"zh": "当前路由不支持 %s 对应的操作。",
	},
	"gateway.gemini.invalid_api_key": {
		"en": "Invalid API key",
		"zh": "API 密钥无效。",
	},
	"gateway.gemini.batch_service_missing": {
		"en": "Gemini batch service not configured",
		"zh": "Gemini Batch 服务尚未配置。",
	},
	"gateway.gemini.unsupported_platform": {
		"en": "Gemini protocol is not supported for this platform",
		"zh": "当前平台不支持 Gemini 协议。",
	},
	"gateway.gemini.missing_model": {
		"en": "Missing model in URL",
		"zh": "URL 中缺少模型参数。",
	},
	"gateway.gemini.model_not_found": {
		"en": "Model not found",
		"zh": "未找到模型。",
	},
	"gateway.gemini.model_action_path_missing": {
		"en": "Gemini model action path is missing",
		"zh": "Gemini 模型 action 路径缺失。",
	},
	"gateway.gemini.model_action_path_invalid": {
		"en": "Gemini model action path is invalid",
		"zh": "Gemini 模型 action 路径无效。",
	},
	"gateway.gemini.user_context_missing": {
		"en": "User context not found",
		"zh": "未找到用户上下文。",
	},
	"gateway.gemini.group_platform_invalid": {
		"en": "API key group platform is not gemini",
		"zh": "当前 API Key 所属分组不是 Gemini 平台。",
	},
	"gateway.gemini.read_body_failed": {
		"en": "Failed to read request body",
		"zh": "读取请求体失败。",
	},
	"gateway.gemini.prepare_body_failed": {
		"en": "Failed to prepare request body",
		"zh": "构造请求体失败。",
	},
	"gateway.gemini.body_empty": {
		"en": "Request body is empty",
		"zh": "请求体不能为空。",
	},
	"gateway.gemini.pending_requests": {
		"en": "Too many pending requests, please retry later",
		"zh": "排队中的请求过多，请稍后重试。",
	},
	"gateway.gemini.request_failed": {
		"en": "Request failed",
		"zh": "请求失败。",
	},
	"gateway.gemini.request_body_too_large": {
		"en": "Request body too large, limit is %s",
		"zh": "请求体过大，限制为 %s。",
	},
	"gateway.gemini.group_exhausted": {
		"en": "All accounts in the selected group have been exhausted",
		"zh": "所选分组中的账户已全部耗尽。",
	},
	"gateway.gemini.subscription_required": {
		"en": "Active subscription required for this group",
		"zh": "当前分组需要有效订阅。",
	},
	"gateway.gemini.invalid_group_binding": {
		"en": "Selected API key group binding is invalid",
		"zh": "所选 API Key 的分组绑定无效。",
	},
	"gateway.gemini.billing_service_unavailable": {
		"en": "Billing service temporarily unavailable, please retry later",
		"zh": "计费服务暂时不可用，请稍后重试。",
	},
	"gateway.gemini.files_path_unsupported": {
		"en": "Unsupported Gemini Files path",
		"zh": "不支持当前 Gemini Files 路径。",
	},
	"gateway.gemini.batch_path_unsupported": {
		"en": "Unsupported Gemini Batch path",
		"zh": "不支持当前 Gemini Batch 路径。",
	},
	"gateway.gemini.file_download_not_found": {
		"en": "File not found",
		"zh": "未找到文件。",
	},
	"gateway.gemini.archive_batch_not_found": {
		"en": "Archive batch not found",
		"zh": "未找到归档批任务。",
	},
	"gateway.gemini.archive_file_not_found": {
		"en": "Archive file not found",
		"zh": "未找到归档文件。",
	},
	"gateway.gemini.vertex_batch_path_invalid": {
		"en": "Vertex batch request path is invalid",
		"zh": "Vertex Batch 请求路径无效。",
	},
	"gateway.gemini.batch_no_account": {
		"en": "No available Google batch accounts",
		"zh": "当前没有可用的 Google Batch 账户。",
	},
	"gateway.gemini.no_available_group": {
		"en": "No available group for this request",
		"zh": "当前请求没有可用分组。",
	},
	"gateway.gemini.rate_limit_exceeded": {
		"en": "Gateway rate limit exceeded, please retry later",
		"zh": "网关已触发限流，请稍后重试。",
	},
	"gateway.gemini.channel_model_not_allowed": {
		"en": "Requested model is not allowed by the bound channel",
		"zh": "绑定通道不允许使用所请求的模型。",
	},
	"gateway.gemini.channel_routing_failed": {
		"en": "Failed to resolve channel routing",
		"zh": "解析通道路由失败。",
	},
	"gateway.gemini.no_available_accounts": {
		"en": "No available Gemini accounts",
		"zh": "没有可用的 Gemini 账号。",
	},
	"gateway.gemini.no_available_accounts_detail": {
		"en": "No available Gemini accounts: %s",
		"zh": "没有可用的 Gemini 账号：%s",
	},
	"gateway.gemini.upstream_failed": {
		"en": "Upstream request failed",
		"zh": "上游请求失败。",
	},
	"gateway.gemini.upstream_empty": {
		"en": "Empty upstream response",
		"zh": "上游响应为空。",
	},
	"gateway.gemini.upstream_auth_failed": {
		"en": "Upstream authentication failed, please contact administrator",
		"zh": "上游认证失败，请联系管理员。",
	},
	"gateway.gemini.upstream_forbidden": {
		"en": "Upstream access forbidden, please contact administrator",
		"zh": "上游访问被拒绝，请联系管理员。",
	},
	"gateway.gemini.upstream_rate_limited": {
		"en": "Upstream rate limit exceeded, please retry later",
		"zh": "上游触发限流，请稍后重试。",
	},
	"gateway.gemini.upstream_overloaded": {
		"en": "Upstream service overloaded, please retry later",
		"zh": "上游服务过载，请稍后重试。",
	},
	"gateway.gemini.upstream_unavailable": {
		"en": "Upstream service temporarily unavailable",
		"zh": "上游服务暂时不可用。",
	},
	"compat.anthropic.system_invalid": {
		"en": "system must be a string or an array of text blocks",
		"zh": "system 必须是字符串或 text block 数组。",
	},
	"compat.anthropic.tool_choice_invalid": {
		"en": "tool_choice must be an object with a valid type",
		"zh": "tool_choice 必须是带有效 type 的对象。",
	},
	"compat.anthropic.message_content_invalid": {
		"en": "Anthropic message content must be a string or an array of content blocks",
		"zh": "Anthropic message content 必须是字符串或 content block 数组。",
	},
	"compat.chat.user_content_invalid": {
		"en": "user message content must be a string or an array of content parts",
		"zh": "user message content 必须是字符串或 content part 数组。",
	},
	"compat.chat.string_content_invalid": {
		"en": "message content must be a JSON string on this compatibility path",
		"zh": "当前兼容路径要求 message content 为 JSON 字符串。",
	},
	"compat.chat.function_call_invalid": {
		"en": "function_call must be a string or an object with a name",
		"zh": "function_call 必须是字符串或带 name 的对象。",
	},
	"compat.gemini.messages_invalid": {
		"en": "messages must be an array",
		"zh": "messages 必须是数组。",
	},
	"compat.gemini.url_context_unsupported": {
		"en": "urlContext is only available on Gemini API accounts, not Vertex Gemini channels",
		"zh": "urlContext 仅支持 Gemini API 账号，不支持 Vertex Gemini 通道。",
	},
	"compat.gemini.thinking_conflict": {
		"en": "thinkingLevel and thinkingBudget cannot be used together for Gemini 3 models",
		"zh": "Gemini 3 模型不能同时使用 thinkingLevel 和 thinkingBudget。",
	},
	"compat.gemini.minimal_thinking_unsupported": {
		"en": "thinkingLevel MINIMAL is only supported by gemini-3-flash-preview and gemini-3.1-flash-lite-preview",
		"zh": "thinkingLevel=MINIMAL 仅支持 gemini-3-flash-preview 和 gemini-3.1-flash-lite-preview。",
	},
	"compat.gemini.thinking_level_unsupported": {
		"en": "thinkingLevel is only supported by Gemini 3 thinking models",
		"zh": "thinkingLevel 仅支持 Gemini 3 thinking 模型。",
	},
	"compat.gemini.reasoning_none_unsupported": {
		"en": "reasoning_effort=none is only supported by gemini-3-flash-preview and gemini-3.1-flash-lite-preview",
		"zh": "reasoning_effort=none 仅支持 gemini-3-flash-preview 和 gemini-3.1-flash-lite-preview。",
	},
	"compat.gemini.media_resolution_invalid": {
		"en": "mediaResolution only supports LOW, MEDIUM, or HIGH on the current Gemini route",
		"zh": "当前 Gemini 路由仅支持 LOW、MEDIUM 或 HIGH 的 mediaResolution。",
	},
	"compat.gemini.media_resolution_unsupported": {
		"en": "mediaResolution is only supported by Gemini 3 models",
		"zh": "mediaResolution 仅支持 Gemini 3 模型。",
	},
	"compat.runtime.registry_missing": {
		"en": "compat conversion is not configured for this route",
		"zh": "当前路由未配置对应的 compat 转换。",
	},
	"admin.account.invalid_request": {
		"en": "Invalid request: %s",
		"zh": "请求无效：%s",
	},
	"admin.account.invalid_id": {
		"en": "Invalid account ID",
		"zh": "账号 ID 无效。",
	},
	"admin.account.deleted": {
		"en": "Account deleted successfully",
		"zh": "账号已删除。",
	},
	"admin.account.rate_multiplier_invalid": {
		"en": "rate_multiplier must be >= 0",
		"zh": "rate_multiplier 必须大于等于 0。",
	},
	"admin.account.no_updates": {
		"en": "No updates provided",
		"zh": "未提供可更新内容。",
	},
	"admin.account.account_ids_required": {
		"en": "account_ids is required",
		"zh": "必须提供 account_ids。",
	},
	"admin.account.test_service_missing": {
		"en": "Account test service is not configured",
		"zh": "账号测试服务尚未配置。",
	},
	"admin.account.model_required_for_manual_mode": {
		"en": "model_id is required when model_input_mode is not auto",
		"zh": "当 model_input_mode 不是 auto 时，必须提供 model_id。",
	},
	"admin.account.ids_delete_all_conflict": {
		"en": "ids and delete_all cannot be provided together",
		"zh": "ids 和 delete_all 不能同时提供。",
	},
	"admin.account.ids_or_delete_all_required": {
		"en": "either ids or delete_all=true is required",
		"zh": "必须提供 ids，或将 delete_all 设为 true。",
	},
	"admin.account.gateway_protocol_invalid": {
		"en": "Invalid gateway protocol",
		"zh": "协议网关协议无效。",
	},
	"admin.account.crs_sync_missing": {
		"en": "CRS sync service is not configured",
		"zh": "CRS 同步服务尚未配置。",
	},
	"admin.account.rate_limit_service_missing": {
		"en": "Rate limit service is not configured",
		"zh": "限流服务尚未配置。",
	},
	"admin.account.usage_service_missing": {
		"en": "Account usage service is not configured",
		"zh": "账号用量服务尚未配置。",
	},
	"admin.account.invalid_days": {
		"en": "Invalid days",
		"zh": "days 参数无效。",
	},
	"admin.account.invalid_source": {
		"en": "Invalid source",
		"zh": "source 参数无效。",
	},
	"admin.account.invalid_group_filter": {
		"en": "Invalid group filter",
		"zh": "分组筛选条件无效。",
	},
	"admin.account.temp_unsched_cleared": {
		"en": "Temporary unschedulable status cleared",
		"zh": "已清除临时不可调度状态。",
	},
	"admin.account.model_import_service_missing": {
		"en": "Account model import service unavailable",
		"zh": "账号模型导入服务不可用。",
	},
	"admin.account.not_found": {
		"en": "Account not found",
		"zh": "账号不存在。",
	},
	"admin.account.mixed_channel_warning": {
		"en": "mixed_channel_warning: Group '%s' contains both %s and %s accounts. Using mixed channels in the same context may cause thinking block signature validation issues, which will fallback to non-thinking mode for historical messages.",
		"zh": "mixed_channel_warning：分组“%s”同时包含 %s 与 %s 账号。在同一上下文中混用通道可能触发 thinking block signature 校验问题，并让历史消息回退到非 thinking 模式。",
	},
}

func localizeMessage(c *gin.Context, message string, reason string) string {
	if c == nil {
		return message
	}
	locale := detectLocale(contextAcceptLanguage(c))
	if translations, ok := localizedReasonMessages[strings.TrimSpace(reason)]; ok {
		if translated := strings.TrimSpace(translations[locale]); translated != "" {
			return translated
		}
		if translated := strings.TrimSpace(translations["en"]); translated != "" {
			protocolruntime.RecordLocalizationFallback("reason:en")
			slog.Info("localized_message_fallback", "reason", strings.TrimSpace(reason), "locale", locale, "fallback", "en")
			return translated
		}
	}
	if strings.TrimSpace(reason) != "" {
		protocolruntime.RecordLocalizationFallback("reason:original_message")
		slog.Info("localized_message_fallback", "reason", strings.TrimSpace(reason), "locale", locale, "fallback", "original_message")
	}
	return message
}

func LocalizedMessage(c *gin.Context, messageKey string, fallback string, args ...any) string {
	template, ok := localizedMessageTemplate(c, messageKey)
	if !ok {
		template = fallback
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

func localizedMessageTemplate(c *gin.Context, messageKey string) (string, bool) {
	translations, ok := localizedMessageKeys[strings.TrimSpace(messageKey)]
	if !ok {
		return "", false
	}

	locale := "en"
	if c != nil {
		locale = detectLocale(contextAcceptLanguage(c))
	}

	template := strings.TrimSpace(translations[locale])
	if template == "" {
		template = strings.TrimSpace(translations["en"])
		if template != "" {
			protocolruntime.RecordLocalizationFallback("message_key:en")
			slog.Info("localized_message_fallback", "message_key", strings.TrimSpace(messageKey), "locale", locale, "fallback", "en")
		}
	}
	if template == "" {
		protocolruntime.RecordLocalizationFallback("message_key:missing_key")
		slog.Info("localized_message_fallback", "message_key", strings.TrimSpace(messageKey), "locale", locale, "fallback", "missing_key")
		return "", false
	}
	return template, true
}

func LocalizedCompatErrorMessage(c *gin.Context, err error) (string, bool) {
	if err == nil {
		return "", false
	}

	var compatErr *apicompat.CompatError
	if !errors.As(err, &compatErr) || compatErr == nil {
		return "", false
	}

	fallback := strings.TrimSpace(compatErr.Message)
	if fallback == "" && compatErr.Err != nil {
		fallback = compatErr.Err.Error()
	}
	if fallback == "" {
		fallback = compatErr.Reason
	}
	if template, ok := localizedMessageTemplate(c, compatErr.MessageKey); ok {
		return template, true
	}
	return fallback, true
}

func HasLocalizedReasonMessage(reason string) bool {
	translations, ok := localizedReasonMessages[strings.TrimSpace(reason)]
	if !ok {
		return false
	}
	return strings.TrimSpace(translations["en"]) != "" && strings.TrimSpace(translations["zh"]) != ""
}

func HasLocalizedMessageKey(messageKey string) bool {
	translations, ok := localizedMessageKeys[strings.TrimSpace(messageKey)]
	if !ok {
		return false
	}
	return strings.TrimSpace(translations["en"]) != "" && strings.TrimSpace(translations["zh"]) != ""
}

func detectLocale(header string) string {
	for _, part := range strings.Split(header, ",") {
		language := strings.ToLower(strings.TrimSpace(part))
		if idx := strings.Index(language, ";"); idx >= 0 {
			language = strings.TrimSpace(language[:idx])
		}
		if strings.HasPrefix(language, "zh") {
			return "zh"
		}
		if language != "" {
			return "en"
		}
	}
	return "en"
}

func contextAcceptLanguage(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.GetHeader("Accept-Language")
}
