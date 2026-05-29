package service

type SyncFromCRSInput struct {
	BaseURL            string
	Username           string
	Password           string
	SyncProxies        bool
	SelectedAccountIDs []string // if non-empty, only create new accounts with these CRS IDs
}

type SyncFromCRSItemResult struct {
	CRSAccountID string `json:"crs_account_id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Action       string `json:"action"` // created/updated/failed/skipped
	Error        string `json:"error,omitempty"`
}

type SyncFromCRSResult struct {
	Created int                     `json:"created"`
	Updated int                     `json:"updated"`
	Skipped int                     `json:"skipped"`
	Failed  int                     `json:"failed"`
	Items   []SyncFromCRSItemResult `json:"items"`
}

type crsLoginResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token"`
	Message  string `json:"message"`
	Error    string `json:"error"`
	Username string `json:"username"`
}

type crsExportResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
	Data    struct {
		ExportedAt              string                      `json:"exportedAt"`
		ClaudeAccounts          []crsClaudeAccount          `json:"claudeAccounts"`
		ClaudeConsoleAccounts   []crsConsoleAccount         `json:"claudeConsoleAccounts"`
		OpenAIOAuthAccounts     []crsOpenAIOAuthAccount     `json:"openaiOAuthAccounts"`
		OpenAIResponsesAccounts []crsOpenAIResponsesAccount `json:"openaiResponsesAccounts"`
		GeminiOAuthAccounts     []crsGeminiOAuthAccount     `json:"geminiOAuthAccounts"`
		GeminiAPIKeyAccounts    []crsGeminiAPIKeyAccount    `json:"geminiApiKeyAccounts"`
	} `json:"data"`
}

type crsProxy struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type crsClaudeAccount struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"`
	AuthType    string         `json:"authType"` // oauth/setup-token
	IsActive    bool           `json:"isActive"`
	Schedulable bool           `json:"schedulable"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Proxy       *crsProxy      `json:"proxy"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
}

type crsConsoleAccount struct {
	Kind               string         `json:"kind"`
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Platform           string         `json:"platform"`
	IsActive           bool           `json:"isActive"`
	Schedulable        bool           `json:"schedulable"`
	Priority           int            `json:"priority"`
	Status             string         `json:"status"`
	MaxConcurrentTasks int            `json:"maxConcurrentTasks"`
	Proxy              *crsProxy      `json:"proxy"`
	Credentials        map[string]any `json:"credentials"`
}

type crsOpenAIResponsesAccount struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"`
	IsActive    bool           `json:"isActive"`
	Schedulable bool           `json:"schedulable"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Proxy       *crsProxy      `json:"proxy"`
	Credentials map[string]any `json:"credentials"`
}

type crsOpenAIOAuthAccount struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"`
	AuthType    string         `json:"authType"` // oauth
	IsActive    bool           `json:"isActive"`
	Schedulable bool           `json:"schedulable"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Proxy       *crsProxy      `json:"proxy"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
}

type crsGeminiOAuthAccount struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"`
	AuthType    string         `json:"authType"` // oauth
	IsActive    bool           `json:"isActive"`
	Schedulable bool           `json:"schedulable"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Proxy       *crsProxy      `json:"proxy"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
}

type crsGeminiAPIKeyAccount struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"`
	IsActive    bool           `json:"isActive"`
	Schedulable bool           `json:"schedulable"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Proxy       *crsProxy      `json:"proxy"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
}

// PreviewFromCRSResult contains the preview of accounts from CRS before sync.
type PreviewFromCRSResult struct {
	NewAccounts      []CRSPreviewAccount `json:"new_accounts"`
	ExistingAccounts []CRSPreviewAccount `json:"existing_accounts"`
}

// CRSPreviewAccount represents a single account in the preview result.
type CRSPreviewAccount struct {
	CRSAccountID string `json:"crs_account_id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	Type         string `json:"type"`
}
