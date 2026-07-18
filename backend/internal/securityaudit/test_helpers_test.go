package securityaudit

import (
	"context"
	"errors"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type memorySettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newMemorySettingRepo() *memorySettingRepo {
	return &memorySettingRepo{values: map[string]string{}}
}

func (r *memorySettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (r *memorySettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *memorySettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *memorySettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, err := r.GetValue(context.Background(), key); err == nil {
			out[key] = value
		}
	}
	return out, nil
}

func (r *memorySettingRepo) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := r.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *memorySettingRepo) GetAll(context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *memorySettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

type plainEncryptor struct{}

func (plainEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (plainEncryptor) Decrypt(value string) (string, error) {
	if len(value) < 4 || value[:4] != "enc:" {
		return "", errors.New("ciphertext invalid")
	}
	return value[4:], nil
}

func newTestPromptService(t interface{ Fatalf(string, ...any) }, req UpdateConfigRequest, scanner PromptScanner, repo promptRepository) *PromptService {
	ctx := context.Background()
	settings := newMemorySettingRepo()
	manager := NewConfigManager(settings, plainEncryptor{})
	if req.Strategy == "" {
		req.Strategy = DefaultStrategy
	}
	if req.WorkerCount == 0 {
		req.WorkerCount = DefaultWorkerCount
	}
	if req.QueueCapacity == 0 {
		req.QueueCapacity = DefaultQueueCapacity
	}
	if _, err := manager.Save(ctx, req, 7); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if scanner == nil {
		scanner = &fakeScanner{result: passResult()}
	}
	if repo == nil {
		repo = &fakePromptRepo{}
	}
	return NewPromptService(manager, repo, newMemoryPayloadStore(), scanner, NewAtomicMetrics())
}

func passResult() *NormalizedResult {
	return &NormalizedResult{
		Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, Safety: "Safe",
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		ScannerBackend: "qwen3guard-openai", ScannerVersion: DefaultGuardModel,
		PolicyID: DefaultStrategy, PolicyVersion: 1,
	}
}

func blockResult() *NormalizedResult {
	return &NormalizedResult{
		Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock, Safety: "Unsafe",
		Categories: []string{"jailbreak"}, MatchedScanners: []string{"jailbreak"},
		ScannerScores: map[string]float64{"jailbreak": 1}, ScannerEvidence: map[string]string{"jailbreak": "Jailbreak"},
		ScannerBackend: "qwen3guard-openai", ScannerVersion: DefaultGuardModel,
		PolicyID: DefaultStrategy, PolicyVersion: 1,
	}
}

func enabledConfig(blocking, storePass bool) UpdateConfigRequest {
	return UpdateConfigRequest{
		Enabled: true, BlockingEnabled: blocking, StorePassEvents: storePass,
		Strategy: DefaultStrategy, WorkerCount: 1, QueueCapacity: 8, Scanners: AllScannerIDs,
		AllGroups: true,
		Endpoints: []UpdateEndpoint{{
			ID: "primary", Name: "Primary", Protocol: "openai_compatible",
			BaseURL: "https://guard.example.com/v1", Model: DefaultGuardModel,
			TimeoutMS: DefaultTimeoutMS, InputLimit: DefaultInputLimit, Enabled: true,
		}},
	}
}
