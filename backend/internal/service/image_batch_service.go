package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
)

type ImageBatchService struct {
	repo           ImageBatchRepository
	gatewayService *GatewayService
	apiKeyService  *APIKeyService
	settingService *SettingService
	forwarder      ImageBatchGoogleForwarder
	cfg            *config.Config
	stopCh         chan struct{}
	downloadMu     sync.Mutex
	downloadSlots  map[string]chan struct{}
	downloadLimits map[string]int
}

func NewImageBatchService(repo ImageBatchRepository, gatewayService *GatewayService, apiKeyService *APIKeyService, settingService *SettingService, forwarder *GeminiNativeGatewayService, cfg *config.Config) *ImageBatchService {
	return &ImageBatchService{
		repo:           repo,
		gatewayService: gatewayService,
		apiKeyService:  apiKeyService,
		settingService: settingService,
		forwarder:      forwarder,
		cfg:            cfg,
		stopCh:         make(chan struct{}),
	}
}

func (s *ImageBatchService) Submit(ctx context.Context, apiKey *APIKey, req ImageBatchSubmitRequest, idempotencyKey string) (*ImageBatchJobResponse, error) {
	if s == nil || s.repo == nil || s.gatewayService == nil || apiKey == nil || apiKey.User == nil {
		return nil, ErrImageBatchUnavailable
	}
	if !s.imageBatchGloballyEnabled(ctx) {
		return nil, ErrImageBatchUnavailable
	}
	groupID := derefInt64(apiKey.GroupID)
	if groupID <= 0 {
		return nil, ErrImageBatchUnavailable
	}
	provider := normalizeImageBatchProvider(req.Provider)
	settings, err := s.repo.GetGroupSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !imageBatchGroupAllows(settings, provider, req.Model, len(req.Items)) {
		return nil, ErrImageBatchUnavailable
	}
	platform := imageBatchProviderPlatform(provider)
	if platform == "" {
		return nil, ErrImageBatchInvalidProvider
	}
	entry, ok, err := s.gatewayService.FindAPIKeyPublicModel(ctx, apiKey, platform, req.Model)
	if err != nil {
		return nil, err
	}
	if !ok || entry == nil {
		return nil, ErrImageBatchInvalidModel
	}
	if native, _ := s.gatewayService.resolvePublicImageCapability(ctx, entry); !native {
		return nil, ErrImageBatchInvalidModel
	}
	items, err := normalizeImageBatchItems(req)
	if err != nil {
		return nil, err
	}
	jobID := uuid.NewString()
	fingerprint := imageBatchRequestFingerprint(req, apiKey.ID)
	holdRequestID := ""
	if s.apiKeyService != nil {
		holdCtx := context.WithValue(ctx, ctxkey.ClientRequestID, "image_batch:"+jobID)
		holdCtx = context.WithValue(holdCtx, ctxkey.RequestPayloadHash, fingerprint)
		hold, holdErr := s.apiKeyService.TryReserveRequestBillingHold(holdCtx, apiKey, s.cfg)
		if holdErr != nil {
			return nil, holdErr
		}
		if hold != nil {
			holdRequestID = hold.RequestID
		}
	}
	groupIDPtr := apiKey.GroupID
	job := &ImageBatchJob{
		ID:                 jobID,
		UserID:             apiKey.User.ID,
		APIKeyID:           apiKey.ID,
		GroupID:            groupIDPtr,
		Provider:           provider,
		DisplayModelID:     strings.TrimSpace(entry.PublicID),
		TargetModelID:      strings.TrimSpace(entry.SourceID),
		Size:               normalizeImageBatchSize(req.Size),
		Status:             ImageBatchJobCreated,
		ItemCount:          len(items),
		HoldRequestID:      holdRequestID,
		IdempotencyKeyHash: HashIdempotencyKey(idempotencyKey),
		RequestFingerprint: fingerprint,
		Metadata:           copyStringAnyMap(req.Metadata),
	}
	if job.DisplayModelID == "" {
		job.DisplayModelID = strings.TrimSpace(req.Model)
	}
	if err := s.repo.CreateJob(ctx, job, items); err != nil {
		return nil, err
	}
	recordImageBatchSubmitted()
	_ = s.repo.AddEvent(ctx, job.ID, nil, "submitted", "", string(ImageBatchJobCreated), "", requestIDFromContext(ctx), nil)
	return ImageBatchJobToResponse(job), nil
}

func (s *ImageBatchService) ListJobs(ctx context.Context, userID int64, limit int, before time.Time) ([]ImageBatchJobResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	jobs, err := s.repo.ListJobsForUser(ctx, userID, limit, before)
	if err != nil {
		return nil, err
	}
	out := make([]ImageBatchJobResponse, 0, len(jobs))
	for i := range jobs {
		out = append(out, *ImageBatchJobToResponse(&jobs[i]))
	}
	return out, nil
}

func (s *ImageBatchService) GetJob(ctx context.Context, userID int64, jobID string) (*ImageBatchJobResponse, error) {
	job, err := s.repo.GetJobForUser(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	return ImageBatchJobToResponse(job), nil
}

func (s *ImageBatchService) ListItems(ctx context.Context, userID int64, jobID string, limit int) ([]ImageBatchItemResponse, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	items, err := s.repo.ListItemsForUser(ctx, userID, jobID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ImageBatchItemResponse, 0, len(items))
	for i := range items {
		out = append(out, *ImageBatchItemToResponse(&items[i]))
	}
	return out, nil
}

func (s *ImageBatchService) Cancel(ctx context.Context, userID int64, jobID string) (*ImageBatchJobResponse, error) {
	job, err := s.repo.TryCancelJob(ctx, userID, jobID, "cancelled by user")
	if err != nil {
		return nil, err
	}
	s.releaseJobHoldBestEffort(ctx, job)
	return ImageBatchJobToResponse(job), nil
}

func (s *ImageBatchService) Delete(ctx context.Context, userID int64, jobID string) error {
	return s.repo.SoftDeleteJob(ctx, userID, jobID)
}

func (s *ImageBatchService) DeleteOutputs(ctx context.Context, userID int64, jobID string) error {
	return s.repo.MarkOutputsDeleted(ctx, userID, jobID)
}

func imageBatchRequestFingerprint(req ImageBatchSubmitRequest, apiKeyID int64) string {
	raw, _ := json.Marshal(struct {
		APIKeyID int64                   `json:"api_key_id"`
		Request  ImageBatchSubmitRequest `json:"request"`
	}{APIKeyID: apiKeyID, Request: req})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *ImageBatchService) imageBatchGloballyEnabled(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.settingService == nil {
		return true
	}
	return s.settingService.IsImageBatchEnabled(ctx)
}
