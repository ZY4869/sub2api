package securityaudit

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type storageEndpoint struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	TokenCiphertext string `json:"token_ciphertext,omitempty"`
	TimeoutMS       int    `json:"timeout_ms"`
	InputLimit      int    `json:"input_limit"`
	Enabled         bool   `json:"enabled"`
}

type storageConfig struct {
	Enabled         bool              `json:"enabled"`
	BlockingEnabled bool              `json:"blocking_enabled"`
	StorePassEvents bool              `json:"store_pass_events"`
	Strategy        string            `json:"strategy"`
	WorkerCount     int               `json:"worker_count"`
	QueueCapacity   int               `json:"queue_capacity"`
	Scanners        []string          `json:"scanners"`
	AllGroups       bool              `json:"all_groups"`
	GroupIDs        []int64           `json:"group_ids"`
	Endpoints       []storageEndpoint `json:"endpoints"`
	ConfigVersion   int64             `json:"config_version"`
	UpdatedAt       time.Time         `json:"updated_at"`
	UpdatedBy       int64             `json:"updated_by"`
	ChangeSummary   string            `json:"change_summary"`
}

func defaultStorageConfig() storageConfig {
	return storageConfig{
		Enabled: false, BlockingEnabled: false, StorePassEvents: false, Strategy: DefaultStrategy,
		WorkerCount: DefaultWorkerCount, QueueCapacity: DefaultQueueCapacity,
		Scanners: append([]string(nil), AllScannerIDs...), AllGroups: true, GroupIDs: []int64{},
		Endpoints: []storageEndpoint{}, ConfigVersion: 1,
	}
}

func parseStorageConfig(raw string) (storageConfig, error) {
	cfg := defaultStorageConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return storageConfig{}, err
	}
	normalizeStorageConfig(&cfg)
	if err := validateStorageConfig(cfg); err != nil {
		return storageConfig{}, err
	}
	return cfg, nil
}

func normalizeStorageConfig(cfg *storageConfig) {
	if cfg.ConfigVersion < 1 {
		cfg.ConfigVersion = 1
	}
	if strings.TrimSpace(cfg.Strategy) == "" {
		cfg.Strategy = DefaultStrategy
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = DefaultWorkerCount
	}
	if cfg.QueueCapacity == 0 {
		cfg.QueueCapacity = DefaultQueueCapacity
	}
	cfg.Scanners = canonicalScannerIDs(cfg.Scanners)
	if len(cfg.Scanners) == 0 {
		cfg.Scanners = append([]string(nil), AllScannerIDs...)
	}
	cfg.GroupIDs = canonicalInt64s(cfg.GroupIDs)
	for i := range cfg.Endpoints {
		normalizeEndpoint(&cfg.Endpoints[i])
	}
}

func normalizeEndpoint(ep *storageEndpoint) {
	ep.ID = strings.TrimSpace(ep.ID)
	ep.Name = strings.TrimSpace(ep.Name)
	ep.Protocol = strings.TrimSpace(ep.Protocol)
	if ep.Protocol == "" {
		ep.Protocol = "openai_compatible"
	}
	ep.BaseURL, _ = NormalizeBaseURL(ep.BaseURL)
	ep.Model = strings.TrimSpace(ep.Model)
	if ep.Model == "" {
		ep.Model = DefaultGuardModel
	}
	if ep.TimeoutMS == 0 {
		ep.TimeoutMS = DefaultTimeoutMS
	}
	if ep.InputLimit == 0 {
		ep.InputLimit = DefaultInputLimit
	}
}

func canonicalInt64s(values []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
