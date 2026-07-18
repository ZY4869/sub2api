package securityaudit

type Mode string
type EventDecision string
type RiskLevel string
type Action string

type Request struct {
	RequestID       string
	ClientRequestID string
	UserID          *int64
	Username        string
	UserEmail       string
	APIKeyID        *int64
	APIKeyName      string
	GroupID         *int64
	GroupName       string
	Provider        string
	Endpoint        string
	Protocol        string
	Model           string
	Body            []byte
	Stage           string
}

type EvaluateResult struct {
	Allowed  bool
	Mode     Mode
	Decision EventDecision
	Error    error
	Event    *Event
}

type PromptPayload struct {
	ScanText        string `json:"scan_text"`
	FullPrompt      string `json:"full_prompt,omitempty"`
	ClientRequestID string `json:"client_request_id,omitempty"`
}

type PromptSnapshot struct {
	RequestID          string
	UserID             *int64
	UsernameSnapshot   string
	UserEmailSnapshot  string
	APIKeyID           *int64
	APIKeyNameSnapshot string
	GroupID            *int64
	GroupName          string
	Provider           string
	Endpoint           string
	Protocol           string
	Model              string
	PromptHash         string
	RedactedPreview    string
	FullPrompt         string
	PromptLength       int
	MessageCount       int
	Stage              string
	ScanText           string
}
