package service

import (
	"context"
	"time"
)

const (
	UpstreamResourceKindGeminiFile     = "gemini_file"
	UpstreamResourceKindGeminiBatch    = "gemini_batch"
	UpstreamResourceKindVertexBatchJob = "vertex_batch_job"
	UpstreamResourceKindGeminiCachedContent   = "gemini_cached_content"
	UpstreamResourceKindGeminiFileSearchStore = "gemini_file_search_store"
	UpstreamResourceKindGeminiDocument        = "gemini_document"
	UpstreamResourceKindGeminiOperation       = "gemini_operation"
	UpstreamResourceKindGeminiInteraction     = "gemini_interaction"

	UpstreamProviderAIStudio = "ai_studio"
	UpstreamProviderVertexAI = "vertex_ai"

	GeminiBatchCapabilityNone     = "none"
	GeminiBatchCapabilityAIStudio = "ai_studio_batch"
	GeminiBatchCapabilityVertex   = "vertex_batch"
)

type UpstreamResourceBinding struct {
	ID             int64
	ResourceKind   string
	ResourceName   string
	ProviderFamily string
	AccountID      int64
	APIKeyID       *int64
	GroupID        *int64
	UserID         *int64
	MetadataJSON   map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type UpstreamResourceBindingRepository interface {
	Upsert(ctx context.Context, binding *UpstreamResourceBinding) error
	Get(ctx context.Context, resourceKind, resourceName string) (*UpstreamResourceBinding, error)
	GetByNames(ctx context.Context, resourceKind string, resourceNames []string) ([]*UpstreamResourceBinding, error)
	SoftDelete(ctx context.Context, resourceKind, resourceName string) error
}
