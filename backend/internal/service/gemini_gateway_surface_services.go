package service

import (
	"context"
)

type GeminiNativeGatewayService struct {
	*GeminiMessagesCompatService
}

func NewGeminiNativeGatewayService(compat *GeminiMessagesCompatService) *GeminiNativeGatewayService {
	return &GeminiNativeGatewayService{GeminiMessagesCompatService: compat}
}

func (s *GeminiNativeGatewayService) ForwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	return s.forwardGeminiPassthrough(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardGoogleFiles(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardGoogleFiles(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardGoogleFileDownload(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardGoogleFileDownload(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardGoogleBatches(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardGoogleBatches(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardVertexBatchPredictionJobs(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardVertexBatchPredictionJobs(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardSimplifiedVertexBatchPredictionJobs(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardSimplifiedVertexBatchPredictionJobs(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardGoogleArchiveBatch(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardGoogleArchiveBatch(ctx, input)
}

func (s *GeminiNativeGatewayService) ForwardGoogleArchiveFileDownload(ctx context.Context, input GoogleBatchForwardInput) (GoogleBatchUpstreamResult, *Account, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil, nil
	}
	return s.GeminiMessagesCompatService.ForwardGoogleArchiveFileDownload(ctx, input)
}

type GeminiCompatGatewayService struct {
	*GeminiMessagesCompatService
}

func NewGeminiCompatGatewayService(compat *GeminiMessagesCompatService) *GeminiCompatGatewayService {
	return &GeminiCompatGatewayService{GeminiMessagesCompatService: compat}
}

func (s *GeminiCompatGatewayService) ForwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	return s.forwardGeminiPassthrough(ctx, input)
}

type GeminiLiveGatewayService struct {
	*GeminiMessagesCompatService
}

func NewGeminiLiveGatewayService(compat *GeminiMessagesCompatService) *GeminiLiveGatewayService {
	return &GeminiLiveGatewayService{GeminiMessagesCompatService: compat}
}

func (s *GeminiLiveGatewayService) ForwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	return s.forwardGeminiPassthrough(ctx, input)
}

type GeminiInteractionsGatewayService struct {
	*GeminiMessagesCompatService
}

func NewGeminiInteractionsGatewayService(compat *GeminiMessagesCompatService) *GeminiInteractionsGatewayService {
	return &GeminiInteractionsGatewayService{GeminiMessagesCompatService: compat}
}

func (s *GeminiInteractionsGatewayService) ForwardGeminiPassthrough(ctx context.Context, input GeminiPublicPassthroughInput) (*GeminiPublicPassthroughOutput, error) {
	if s == nil || s.GeminiMessagesCompatService == nil {
		return nil, nil
	}
	return s.forwardGeminiPassthrough(ctx, input)
}
