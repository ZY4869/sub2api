package securityaudit

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func activeFromStorage(cfg storageConfig, encryptor service.SecretEncryptor) (ActiveConfig, error) {
	active := ActiveConfig{
		Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled, StorePassEvents: cfg.StorePassEvents,
		Strategy: cfg.Strategy, WorkerCount: cfg.WorkerCount, QueueCapacity: cfg.QueueCapacity,
		Scanners: append([]string(nil), cfg.Scanners...), AllGroups: cfg.AllGroups,
		GroupIDs: append([]int64(nil), cfg.GroupIDs...), ConfigVersion: cfg.ConfigVersion,
		UpdatedAt: cfg.UpdatedAt, UpdatedBy: cfg.UpdatedBy, ChangeSummary: cfg.ChangeSummary,
	}
	for _, endpoint := range cfg.Endpoints {
		token, err := decryptEndpointToken(endpoint, encryptor)
		if err != nil {
			return ActiveConfig{}, err
		}
		active.Endpoints = append(active.Endpoints, ActiveEndpoint{
			ID: endpoint.ID, Name: endpoint.Name, Protocol: endpoint.Protocol, BaseURL: endpoint.BaseURL,
			Model: endpoint.Model, Token: token, TimeoutMS: endpoint.TimeoutMS,
			InputLimit: endpoint.InputLimit, Enabled: endpoint.Enabled,
		})
	}
	return active, nil
}

func decryptEndpointToken(endpoint storageEndpoint, encryptor service.SecretEncryptor) (string, error) {
	if endpoint.TokenCiphertext == "" {
		return "", nil
	}
	if encryptor == nil {
		return "", errors.New("prompt audit encryptor unavailable")
	}
	plain, err := encryptor.Decrypt(endpoint.TokenCiphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt prompt audit endpoint %s: %w", endpoint.ID, err)
	}
	return plain, nil
}

func publicFromStorage(cfg storageConfig) PublicConfig {
	endpoints := make([]PublicEndpoint, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		endpoints = append(endpoints, publicEndpointFromStorage(ep))
	}
	return PublicConfig{
		Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled, StorePassEvents: cfg.StorePassEvents,
		EffectiveMode: ActiveConfig{Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled}.EffectiveMode(),
		Strategy:      cfg.Strategy, WorkerCount: cfg.WorkerCount, QueueCapacity: cfg.QueueCapacity,
		Scanners: append([]string(nil), cfg.Scanners...), AllGroups: cfg.AllGroups,
		GroupIDs: append([]int64(nil), cfg.GroupIDs...), Endpoints: endpoints,
		ConfigVersion: cfg.ConfigVersion, UpdatedAt: cfg.UpdatedAt,
		UpdatedBy: cfg.UpdatedBy, ChangeSummary: cfg.ChangeSummary,
	}
}

func publicEndpointFromStorage(ep storageEndpoint) PublicEndpoint {
	status := "missing"
	if strings.TrimSpace(ep.TokenCiphertext) != "" {
		status = "configured"
	}
	return PublicEndpoint{
		ID: ep.ID, Name: ep.Name, Protocol: ep.Protocol, BaseURL: ep.BaseURL, Model: ep.Model,
		TimeoutMS: ep.TimeoutMS, InputLimit: ep.InputLimit, Enabled: ep.Enabled,
		HasToken: status == "configured", TokenStatus: status,
	}
}

func (cfg ActiveConfig) EffectiveMode() Mode {
	if !cfg.Enabled {
		return ModeOff
	}
	if cfg.BlockingEnabled {
		return ModeBlocking
	}
	return ModeAsync
}

func (cfg ActiveConfig) IncludesGroup(groupID *int64) bool {
	if cfg.AllGroups {
		return true
	}
	if groupID == nil {
		return false
	}
	i := sort.Search(len(cfg.GroupIDs), func(i int) bool { return cfg.GroupIDs[i] >= *groupID })
	return i < len(cfg.GroupIDs) && cfg.GroupIDs[i] == *groupID
}

func (cfg ActiveConfig) EnabledEndpoints() []ActiveEndpoint {
	out := make([]ActiveEndpoint, 0, len(cfg.Endpoints))
	for _, endpoint := range cfg.Endpoints {
		if endpoint.Enabled {
			out = append(out, endpoint)
		}
	}
	return out
}
