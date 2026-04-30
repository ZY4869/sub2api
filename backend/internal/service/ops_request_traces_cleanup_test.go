package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestCleanupRequestTraces_FilterRequiresAtLeastOneCondition(t *testing.T) {
	repo := &opsRepoMock{}
	svc := &OpsService{opsRepo: repo}

	_, err := svc.CleanupRequestTraces(context.Background(), OpsRequestTraceCleanupModeFilter, nil, 123)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if got := infraerrors.Reason(err); got != "OPS_REQUEST_TRACE_CLEANUP_FILTER_REQUIRED" {
		t.Fatalf("Reason(err) = %q, want %q", got, "OPS_REQUEST_TRACE_CLEANUP_FILTER_REQUIRED")
	}
}

func TestCleanupRequestTraces_FilterAllowsTimeRangeOnly(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-30 * time.Minute)

	var gotFilter *OpsRequestTraceFilter
	repo := &opsRepoMock{
		DeleteRequestTracesFn: func(ctx context.Context, filter *OpsRequestTraceFilter) (OpsRequestTraceDeleteCounts, error) {
			gotFilter = filter
			return OpsRequestTraceDeleteCounts{DeletedTraces: 3, DeletedAudits: 4}, nil
		},
	}
	svc := &OpsService{opsRepo: repo}

	filter := &OpsRequestTraceFilter{
		StartTime: &start,
		EndTime:   &now,
		Page:      2,
		PageSize:  20,
		Sort:      "created_at_desc",
		Limit:     10,
	}

	res, err := svc.CleanupRequestTraces(context.Background(), OpsRequestTraceCleanupModeFilter, filter, 123)
	if err != nil {
		t.Fatalf("CleanupRequestTraces() error = %v", err)
	}
	if res == nil {
		t.Fatalf("expected result, got nil")
	}
	if res.Mode != OpsRequestTraceCleanupModeFilter {
		t.Fatalf("Mode = %q, want %q", res.Mode, OpsRequestTraceCleanupModeFilter)
	}
	if res.DeletedTraces != 3 || res.DeletedAudits != 4 {
		t.Fatalf("deleted counts = traces=%d audits=%d, want traces=3 audits=4", res.DeletedTraces, res.DeletedAudits)
	}
	if gotFilter == nil {
		t.Fatalf("expected repo.DeleteRequestTraces to be called")
	}
	if gotFilter.Page != 0 || gotFilter.PageSize != 0 || gotFilter.Sort != "" || gotFilter.Limit != 0 {
		t.Fatalf("expected normalized filter to zero paging/sort/limit, got page=%d page_size=%d sort=%q limit=%d", gotFilter.Page, gotFilter.PageSize, gotFilter.Sort, gotFilter.Limit)
	}
	if gotFilter.StartTime == nil || gotFilter.EndTime == nil {
		t.Fatalf("expected StartTime/EndTime to be preserved")
	}
}

func TestCleanupRequestTraces_ExpiredReturnsCutoffAndDeletes(t *testing.T) {
	var gotCutoff time.Time
	var gotBatchSize int
	repo := &opsRepoMock{
		DeleteExpiredRequestTracesFn: func(ctx context.Context, cutoff time.Time, batchSize int) (OpsRequestTraceDeleteCounts, error) {
			gotCutoff = cutoff
			gotBatchSize = batchSize
			return OpsRequestTraceDeleteCounts{DeletedTraces: 10, DeletedAudits: 20}, nil
		},
	}

	svc := &OpsService{
		opsRepo: repo,
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					RetentionDays: 7,
				},
			},
		},
	}

	startedAt := time.Now().UTC()
	res, err := svc.CleanupRequestTraces(context.Background(), OpsRequestTraceCleanupModeExpired, nil, 456)
	if err != nil {
		t.Fatalf("CleanupRequestTraces() error = %v", err)
	}
	if res == nil || res.Cutoff == nil {
		t.Fatalf("expected cutoff in result")
	}
	if gotBatchSize != 5000 {
		t.Fatalf("batchSize = %d, want 5000", gotBatchSize)
	}

	// Allow a small skew for time.Now() between cutoff computation and this assertion.
	expectedLatest := startedAt.AddDate(0, 0, -7).Add(5 * time.Second)
	expectedEarliest := startedAt.AddDate(0, 0, -7).Add(-5 * time.Second)
	if gotCutoff.Before(expectedEarliest) || gotCutoff.After(expectedLatest) {
		t.Fatalf("cutoff = %s, want within [%s, %s]", gotCutoff.UTC().Format(time.RFC3339Nano), expectedEarliest.UTC().Format(time.RFC3339Nano), expectedLatest.UTC().Format(time.RFC3339Nano))
	}
	if res.DeletedTraces != 10 || res.DeletedAudits != 20 {
		t.Fatalf("deleted counts = traces=%d audits=%d, want traces=10 audits=20", res.DeletedTraces, res.DeletedAudits)
	}
}
