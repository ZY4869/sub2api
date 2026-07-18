package routes

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
)

type promptAuditRouteRepo struct{}

func (r *promptAuditRouteRepo) CreateJob(context.Context, *securityaudit.Job) error { return nil }
func (r *promptAuditRouteRepo) CreateEvent(context.Context, *securityaudit.Job, *securityaudit.NormalizedResult, string) (*securityaudit.Event, error) {
	return &securityaudit.Event{}, nil
}
func (r *promptAuditRouteRepo) ClaimJobs(context.Context, int) ([]*securityaudit.Job, error) {
	return nil, nil
}
func (r *promptAuditRouteRepo) MarkJobDone(context.Context, int64) error { return nil }
func (r *promptAuditRouteRepo) MarkJobFailed(context.Context, *securityaudit.Job, string, string, bool) error {
	return nil
}
func (r *promptAuditRouteRepo) QueueStats(context.Context) (securityaudit.QueueStats, error) {
	return securityaudit.QueueStats{}, nil
}
func (r *promptAuditRouteRepo) ListEvents(context.Context, securityaudit.EventFilter) (*securityaudit.EventList, error) {
	return &securityaudit.EventList{}, nil
}
func (r *promptAuditRouteRepo) GetEvent(context.Context, int64, bool) (*securityaudit.Event, error) {
	return nil, securityaudit.ErrEventNotFound
}
func (r *promptAuditRouteRepo) DeleteEvent(context.Context, int64) (*securityaudit.DeleteResult, error) {
	return &securityaudit.DeleteResult{}, nil
}
func (r *promptAuditRouteRepo) PreviewDelete(context.Context, securityaudit.EventFilter) (*securityaudit.DeletePreview, error) {
	return &securityaudit.DeletePreview{}, nil
}
func (r *promptAuditRouteRepo) DeleteEventsByFilter(context.Context, securityaudit.EventFilter, int64) (*securityaudit.DeleteResult, error) {
	return &securityaudit.DeleteResult{}, nil
}

type promptAuditRoutePayloadStore struct{}

func (s *promptAuditRoutePayloadStore) Set(context.Context, int64, securityaudit.PromptPayload, time.Duration) error {
	return nil
}
func (s *promptAuditRoutePayloadStore) Get(context.Context, int64) (securityaudit.PromptPayload, error) {
	return securityaudit.PromptPayload{}, nil
}
func (s *promptAuditRoutePayloadStore) Delete(context.Context, int64) error { return nil }
func (s *promptAuditRoutePayloadStore) Ping(context.Context) error          { return nil }
