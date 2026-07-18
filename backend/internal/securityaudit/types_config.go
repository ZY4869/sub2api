package securityaudit

import "time"

type ActiveEndpoint struct {
	ID         string
	Name       string
	Protocol   string
	BaseURL    string
	Model      string
	Token      string
	TimeoutMS  int
	InputLimit int
	Enabled    bool
}

type ActiveConfig struct {
	Enabled         bool
	BlockingEnabled bool
	StorePassEvents bool
	Strategy        string
	WorkerCount     int
	QueueCapacity   int
	Scanners        []string
	AllGroups       bool
	GroupIDs        []int64
	Endpoints       []ActiveEndpoint
	ConfigVersion   int64
	UpdatedAt       time.Time
	UpdatedBy       int64
	ChangeSummary   string
}

type PublicEndpoint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	TimeoutMS   int    `json:"timeout_ms"`
	InputLimit  int    `json:"input_limit"`
	Enabled     bool   `json:"enabled"`
	HasToken    bool   `json:"has_token"`
	TokenStatus string `json:"token_status"`
}

type PublicConfig struct {
	Enabled         bool             `json:"enabled"`
	BlockingEnabled bool             `json:"blocking_enabled"`
	StorePassEvents bool             `json:"store_pass_events"`
	EffectiveMode   Mode             `json:"effective_mode"`
	Strategy        string           `json:"strategy"`
	WorkerCount     int              `json:"worker_count"`
	QueueCapacity   int              `json:"queue_capacity"`
	Scanners        []string         `json:"scanners"`
	AllGroups       bool             `json:"all_groups"`
	GroupIDs        []int64          `json:"group_ids"`
	Endpoints       []PublicEndpoint `json:"endpoints"`
	ConfigVersion   int64            `json:"config_version"`
	UpdatedAt       time.Time        `json:"updated_at"`
	UpdatedBy       int64            `json:"updated_by"`
	ChangeSummary   string           `json:"change_summary"`
}

type UpdateEndpoint struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	Token      string `json:"token,omitempty"`
	ClearToken bool   `json:"clear_token"`
	TimeoutMS  int    `json:"timeout_ms"`
	InputLimit int    `json:"input_limit"`
	Enabled    bool   `json:"enabled"`
}

type UpdateConfigRequest struct {
	ExpectedConfigVersion int64            `json:"expected_config_version"`
	Enabled               bool             `json:"enabled"`
	BlockingEnabled       bool             `json:"blocking_enabled"`
	StorePassEvents       bool             `json:"store_pass_events"`
	Strategy              string           `json:"strategy"`
	WorkerCount           int              `json:"worker_count"`
	QueueCapacity         int              `json:"queue_capacity"`
	Scanners              []string         `json:"scanners"`
	AllGroups             bool             `json:"all_groups"`
	GroupIDs              []int64          `json:"group_ids"`
	Endpoints             []UpdateEndpoint `json:"endpoints"`
}
