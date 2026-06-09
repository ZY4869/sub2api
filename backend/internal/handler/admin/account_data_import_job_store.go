package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	accountImportJobStatusQueued        = "queued"
	accountImportJobStatusRunning       = "running"
	accountImportJobStatusSucceeded     = "succeeded"
	accountImportJobStatusPartialFailed = "partial_failed"
	accountImportJobStatusFailed        = "failed"
	accountImportJobStatusCancelled     = "cancelled"
)

type CreateAccountImportJobResponse struct {
	JobID string `json:"job_id"`
}

type AccountImportJobProgress struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
}

type AccountImportJobSnapshot struct {
	JobID                  string                     `json:"job_id"`
	Status                 string                     `json:"status"`
	Progress               AccountImportJobProgress   `json:"progress"`
	Result                 DataImportResult           `json:"result"`
	CreatedAccountsSummary []DataImportCreatedAccount `json:"created_accounts_summary"`
	Error                  string                     `json:"error,omitempty"`
	CancelRequested        bool                       `json:"cancel_requested"`
	StartedAt              *time.Time                 `json:"started_at,omitempty"`
	FinishedAt             *time.Time                 `json:"finished_at,omitempty"`
	CreatedAt              time.Time                  `json:"created_at"`
	UpdatedAt              time.Time                  `json:"updated_at"`
}

type accountImportJob struct {
	mu              sync.Mutex
	jobID           string
	status          string
	progress        AccountImportJobProgress
	result          DataImportResult
	errorMessage    string
	cancelRequested bool
	cancel          context.CancelFunc
	createdAt       time.Time
	updatedAt       time.Time
	startedAt       *time.Time
	finishedAt      *time.Time
}

type accountImportJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*accountImportJob
}

var defaultAccountImportJobs = newAccountImportJobStore()

func newAccountImportJobStore() *accountImportJobStore {
	return &accountImportJobStore{jobs: make(map[string]*accountImportJob)}
}

func (s *accountImportJobStore) create(total int) (*accountImportJob, error) {
	now := time.Now().UTC()
	job := &accountImportJob{
		jobID:     newAccountImportJobID(),
		status:    accountImportJobStatusQueued,
		progress:  AccountImportJobProgress{Total: total},
		createdAt: now,
		updatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.jobID] = job
	return job, nil
}

func (s *accountImportJobStore) get(jobID string) (*accountImportJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	return job, ok
}

func (j *accountImportJob) snapshot() AccountImportJobSnapshot {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.snapshotLocked()
}

func (j *accountImportJob) snapshotLocked() AccountImportJobSnapshot {
	createdAccounts := append([]DataImportCreatedAccount(nil), j.result.CreatedAccounts...)
	errorsCopy := append([]DataImportError(nil), j.result.Errors...)
	result := j.result
	result.CreatedAccounts = createdAccounts
	result.Errors = errorsCopy
	return AccountImportJobSnapshot{
		JobID:                  j.jobID,
		Status:                 j.status,
		Progress:               j.progress,
		Result:                 result,
		CreatedAccountsSummary: createdAccounts,
		Error:                  j.errorMessage,
		CancelRequested:        j.cancelRequested,
		StartedAt:              j.startedAt,
		FinishedAt:             j.finishedAt,
		CreatedAt:              j.createdAt,
		UpdatedAt:              j.updatedAt,
	}
}

func newAccountImportJobID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
