package securityaudit

const (
	SettingKeyPromptAuditConfig = "prompt_audit_config"
	ConfigInvalidationChannel   = "prompt_audit:config:invalidate"
	DefaultGuardModel           = "sileader/qwen3guard:0.6b"
	DefaultStrategy             = "priority"
	DefaultWorkerCount          = 4
	DefaultQueueCapacity        = 32768
	MaxWorkerCount              = 32
	MaxQueueCapacity            = 100000
	DefaultTimeoutMS            = 3000
	MinTimeoutMS                = 100
	MaxTimeoutMS                = 30000
	DefaultInputLimit           = 12000
	MinInputLimit               = 128
	MaxInputLimit               = 100000
	DefaultPayloadTTLSeconds    = 1800
	DefaultAsyncEnqueueSlots    = 128
	DefaultPromptPreviewRunes   = 96
	DefaultFullPromptRunes      = 65536
)

const (
	ProtocolOpenAIChat        = "openai_chat_completions"
	ProtocolOpenAICompletions = "openai_completions"
	ProtocolOpenAIEmbeddings  = "openai_embeddings"
	ProtocolOpenAIAlphaSearch = "openai_alpha_search"
	ProtocolOpenAIResponses   = "openai_responses"
	ProtocolOpenAIImages      = "openai_images"
	ProtocolAnthropic         = "anthropic_messages"
	ProtocolGemini            = "gemini"
	ProtocolGrokMedia         = "grok_media"
	ProtocolResponsesWS       = "openai_responses_ws"
)

const (
	ModeOff      Mode = "off"
	ModeAsync    Mode = "async_audit"
	ModeBlocking Mode = "blocking"
)

const (
	EventPass     EventDecision = "pass"
	EventFlag     EventDecision = "flag"
	EventCritical EventDecision = "critical"
)

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

const (
	ActionAllow Action = "Allow"
	ActionWarn  Action = "Warn"
	ActionBlock Action = "Block"
)

const (
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusRetry      = "retry"
	JobStatusDone       = "done"
	JobStatusFailed     = "failed"
)

const (
	ErrorCodeBlocked         = "prompt_guard_blocked"
	ErrorCodeUnavailable     = "prompt_guard_unavailable"
	ErrorCodeInvalidResponse = "prompt_guard_invalid_response"
	ErrorCodeConfigConflict  = "prompt_audit_config_conflict"
)
