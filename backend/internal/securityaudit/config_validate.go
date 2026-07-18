package securityaudit

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

func validateStorageConfig(cfg storageConfig) error {
	if cfg.BlockingEnabled && !cfg.Enabled {
		return infraerrors.BadRequest(ErrorCodeConfigConflict, "blocking requires prompt audit to be enabled")
	}
	if cfg.Strategy != DefaultStrategy {
		return infraerrors.BadRequest("prompt_audit_invalid_strategy", "prompt audit strategy must be priority")
	}
	if cfg.WorkerCount < 1 || cfg.WorkerCount > MaxWorkerCount {
		return infraerrors.BadRequest("prompt_audit_invalid_worker_count", "invalid worker count")
	}
	if cfg.QueueCapacity < 1 || cfg.QueueCapacity > MaxQueueCapacity {
		return infraerrors.BadRequest("prompt_audit_invalid_queue_capacity", "invalid queue capacity")
	}
	if cfg.Enabled && len(enabledStorageEndpoints(cfg.Endpoints)) == 0 {
		return infraerrors.BadRequest("prompt_audit_endpoint_required", "at least one endpoint is required")
	}
	if !cfg.AllGroups && len(cfg.GroupIDs) == 0 {
		return infraerrors.BadRequest("prompt_audit_groups_required", "at least one group is required")
	}
	return validateEndpoints(cfg.Endpoints)
}

func validateEndpoints(endpoints []storageEndpoint) error {
	seen := map[string]struct{}{}
	for _, endpoint := range endpoints {
		if err := validateEndpoint(endpoint, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateEndpoint(endpoint storageEndpoint, seen map[string]struct{}) error {
	if endpoint.ID == "" || endpoint.Name == "" {
		return infraerrors.BadRequest("prompt_audit_invalid_endpoint", "endpoint id and name are required")
	}
	if _, ok := seen[endpoint.ID]; ok {
		return infraerrors.BadRequest("prompt_audit_duplicate_endpoint", "endpoint id must be unique")
	}
	seen[endpoint.ID] = struct{}{}
	if endpoint.Protocol != "openai_compatible" {
		return infraerrors.BadRequest("prompt_audit_invalid_endpoint_protocol", "only openai compatible endpoints are supported")
	}
	if _, err := NormalizeBaseURL(endpoint.BaseURL); err != nil {
		return err
	}
	if endpoint.TimeoutMS < MinTimeoutMS || endpoint.TimeoutMS > MaxTimeoutMS {
		return infraerrors.BadRequest("prompt_audit_invalid_timeout", "invalid endpoint timeout")
	}
	if endpoint.InputLimit < MinInputLimit || endpoint.InputLimit > MaxInputLimit {
		return infraerrors.BadRequest("prompt_audit_invalid_input_limit", "invalid endpoint input limit")
	}
	return nil
}

func validateUpdate(req UpdateConfigRequest) error {
	cfg := storageConfig{
		Enabled: req.Enabled, BlockingEnabled: req.BlockingEnabled, StorePassEvents: req.StorePassEvents,
		Strategy: req.Strategy, WorkerCount: req.WorkerCount, QueueCapacity: req.QueueCapacity,
		Scanners: req.Scanners, AllGroups: req.AllGroups, GroupIDs: req.GroupIDs,
	}
	for _, ep := range req.Endpoints {
		cfg.Endpoints = append(cfg.Endpoints, storageEndpoint{
			ID: ep.ID, Name: ep.Name, Protocol: ep.Protocol, BaseURL: ep.BaseURL,
			Model: ep.Model, TimeoutMS: ep.TimeoutMS, InputLimit: ep.InputLimit, Enabled: ep.Enabled,
		})
	}
	normalizeStorageConfig(&cfg)
	return validateStorageConfig(cfg)
}

func enabledStorageEndpoints(values []storageEndpoint) []storageEndpoint {
	out := make([]storageEndpoint, 0, len(values))
	for _, endpoint := range values {
		if endpoint.Enabled {
			out = append(out, endpoint)
		}
	}
	return out
}
