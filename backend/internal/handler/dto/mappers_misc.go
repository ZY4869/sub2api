package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func ContentModerationAuditFromService(audit *service.ContentModerationAudit) *ContentModerationAudit {
	if audit == nil {
		return nil
	}
	return &ContentModerationAudit{
		ID:              audit.ID,
		RequestID:       audit.RequestID,
		ClientRequestID: audit.ClientRequestID,
		UserID:          audit.UserID,
		APIKeyID:        audit.APIKeyID,
		Provider:        audit.Provider,
		Model:           audit.Model,
		SourceEndpoint:  audit.SourceEndpoint,
		ContentHash:     audit.ContentHash,
		ContentSummary:  audit.ContentSummary,
		Hit:             audit.Hit,
		DedupeHit:       audit.DedupeHit,
		ErrorReason:     audit.ErrorReason,
		LatencyMs:       audit.LatencyMs,
		CreatedAt:       audit.CreatedAt.Format(time.RFC3339),
	}
}

func timeToUnixSeconds(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	ts := value.Unix()
	return &ts
}

func SettingFromService(s *service.Setting) *Setting {
	if s == nil {
		return nil
	}
	return &Setting{
		ID:        s.ID,
		Key:       s.Key,
		Value:     s.Value,
		UpdatedAt: s.UpdatedAt,
	}
}
