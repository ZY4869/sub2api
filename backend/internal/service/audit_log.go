package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AuditStatusSuccess = "success"
	AuditStatusDenied  = "denied"
	AuditStatusFailure = "failure"

	DefaultAuditLogRetentionDays = 180
	MinAuditLogRetentionDays     = 1
	MaxAuditLogRetentionDays     = 3650
)

var ErrAuditLogNotFound = infraerrors.NotFound("AUDIT_LOG_NOT_FOUND", "audit log not found")

type AuditLog struct {
	ID          int64          `json:"id"`
	ActorUserID *int64         `json:"actor_user_id,omitempty"`
	ActorRole   string         `json:"actor_role"`
	Action      string         `json:"action"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Status      string         `json:"status"`
	RequestID   string         `json:"request_id"`
	ClientIP    string         `json:"client_ip"`
	UserAgent   string         `json:"user_agent"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type AuditLogFilter struct {
	Page        int
	PageSize    int
	ActorUserID *int64
	Action      string
	TargetType  string
	TargetID    string
	Status      string
	RequestID   string
}

type AuditLogList struct {
	Items    []*AuditLog
	Total    int64
	Page     int
	PageSize int
}

type AuditLogRepository interface {
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, filter *AuditLogFilter) (*AuditLogList, error)
	DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type AuditLogRetentionProvider interface {
	GetAuditLogRetentionDays(ctx context.Context) int
}

type AuditLogService struct {
	repo              AuditLogRepository
	retentionDays     int
	retentionProvider AuditLogRetentionProvider
}

func NewAuditLogService(repo AuditLogRepository) *AuditLogService {
	return &AuditLogService{repo: repo, retentionDays: DefaultAuditLogRetentionDays}
}

func (s *AuditLogService) SetRetentionProvider(provider AuditLogRetentionProvider) {
	if s == nil {
		return
	}
	s.retentionProvider = provider
}

func (s *AuditLogService) Record(ctx context.Context, log *AuditLog) error {
	if s == nil || s.repo == nil || log == nil {
		return nil
	}
	normalized := *log
	normalized.Action = trimAuditValue(normalized.Action, 128)
	normalized.TargetType = trimAuditValue(normalized.TargetType, 64)
	normalized.TargetID = trimAuditValue(normalized.TargetID, 128)
	normalized.ActorRole = trimAuditValue(normalized.ActorRole, 32)
	normalized.Status = normalizeAuditStatus(normalized.Status)
	normalized.RequestID = trimAuditValue(normalized.RequestID, 128)
	normalized.ClientIP = trimAuditValue(normalized.ClientIP, 128)
	normalized.UserAgent = trimAuditValue(normalized.UserAgent, 512)
	normalized.Metadata = RedactAuditMetadata(normalized.Metadata)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = time.Now().UTC()
	}
	return s.repo.CreateAuditLog(ctx, &normalized)
}

func (s *AuditLogService) List(ctx context.Context, filter *AuditLogFilter) (*AuditLogList, error) {
	if s == nil || s.repo == nil {
		return &AuditLogList{Items: []*AuditLog{}, Page: 1, PageSize: 20}, nil
	}
	return s.repo.ListAuditLogs(ctx, normalizeAuditLogFilter(filter))
}

func (s *AuditLogService) CleanupExpired(ctx context.Context) (int64, time.Time, error) {
	retentionDays := DefaultAuditLogRetentionDays
	if s != nil {
		retentionDays = NormalizeAuditLogRetentionDays(s.retentionDays)
		if s.retentionProvider != nil {
			retentionDays = s.retentionProvider.GetAuditLogRetentionDays(ctx)
		}
	}
	retentionDays = NormalizeAuditLogRetentionDays(retentionDays)
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	if s == nil || s.repo == nil {
		return 0, cutoff, nil
	}
	deleted, err := s.repo.DeleteAuditLogsBefore(ctx, cutoff)
	return deleted, cutoff, err
}

func NormalizeAuditLogRetentionDays(days int) int {
	if days <= 0 {
		return DefaultAuditLogRetentionDays
	}
	if days < MinAuditLogRetentionDays {
		return MinAuditLogRetentionDays
	}
	if days > MaxAuditLogRetentionDays {
		return MaxAuditLogRetentionDays
	}
	return days
}

func normalizeAuditLogFilter(filter *AuditLogFilter) *AuditLogFilter {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	out := *filter
	if out.Page <= 0 {
		out.Page = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = 20
	}
	if out.PageSize > 200 {
		out.PageSize = 200
	}
	out.Action = strings.TrimSpace(out.Action)
	out.TargetType = strings.TrimSpace(out.TargetType)
	out.TargetID = strings.TrimSpace(out.TargetID)
	out.Status = strings.TrimSpace(out.Status)
	out.RequestID = strings.TrimSpace(out.RequestID)
	return &out
}

func normalizeAuditStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case AuditStatusDenied:
		return AuditStatusDenied
	case AuditStatusFailure:
		return AuditStatusFailure
	default:
		return AuditStatusSuccess
	}
}

func trimAuditValue(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len([]rune(value)) <= maxLen {
		return value
	}
	return string([]rune(value)[:maxLen])
}

func RedactAuditMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		if isSensitiveAuditKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			out[key] = RedactAuditMetadata(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func isSensitiveAuditKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"token", "secret", "password", "authorization", "cookie", "api_key", "apikey"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
