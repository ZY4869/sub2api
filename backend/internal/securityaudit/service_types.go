package securityaudit

import "context"

type ProbeRequest struct {
	Endpoint UpdateEndpoint `json:"endpoint"`
}

type DeleteByFilterRequest struct {
	Filter            EventFilter `json:"filter"`
	SnapshotMaxID     int64       `json:"snapshot_max_id"`
	FilterHash        string      `json:"filter_hash"`
	ConfirmationToken string      `json:"confirmation_token"`
	Confirm           bool        `json:"confirm"`
}

type deleteConfirmationClaims struct {
	FilterHash    string `json:"filter_hash"`
	SnapshotMaxID int64  `json:"snapshot_max_id"`
	AdminID       int64  `json:"admin_id"`
	ExpiresAt     int64  `json:"exp"`
}

type promptRepository interface {
	CreateJob(ctx context.Context, job *Job) error
	CreateEvent(ctx context.Context, job *Job, result *NormalizedResult, fullPrompt string) (*Event, error)
	ClaimJobs(ctx context.Context, limit int) ([]*Job, error)
	MarkJobDone(ctx context.Context, id int64) error
	MarkJobFailed(ctx context.Context, job *Job, code string, message string, retry bool) error
	QueueStats(ctx context.Context) (QueueStats, error)
	ListEvents(ctx context.Context, filter EventFilter) (*EventList, error)
	GetEvent(ctx context.Context, id int64, includeFullText bool) (*Event, error)
	DeleteEvent(ctx context.Context, id int64) (*DeleteResult, error)
	PreviewDelete(ctx context.Context, filter EventFilter) (*DeletePreview, error)
	DeleteEventsByFilter(ctx context.Context, filter EventFilter, snapshotMaxID int64) (*DeleteResult, error)
}
