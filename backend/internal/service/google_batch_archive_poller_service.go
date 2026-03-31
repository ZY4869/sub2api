package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type GoogleBatchArchivePollerService struct {
	jobRepo         GoogleBatchArchiveJobRepository
	compatService   *GeminiMessagesCompatService
	settingService  *SettingService
	startOnce       sync.Once
	stopOnce        sync.Once
	stopCh          chan struct{}
}

func NewGoogleBatchArchivePollerService(
	jobRepo GoogleBatchArchiveJobRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchivePollerService {
	return &GoogleBatchArchivePollerService{
		jobRepo:        jobRepo,
		compatService:  compatService,
		settingService: settingService,
		stopCh:         make(chan struct{}),
	}
}

func (s *GoogleBatchArchivePollerService) Start() {
	if s == nil || s.jobRepo == nil || s.compatService == nil || s.settingService == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.loop()
		logger.LegacyPrintf("service.google_batch_archive_poller", "[GoogleBatchArchivePoller] started")
	})
}

func (s *GoogleBatchArchivePollerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		logger.LegacyPrintf("service.google_batch_archive_poller", "[GoogleBatchArchivePoller] stopped")
	})
}

func (s *GoogleBatchArchivePollerService) loop() {
	for {
		s.runOnce()
		interval := googleBatchArchivePollInterval(s.compatService.getGoogleBatchArchiveSettings(context.Background()))
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-s.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func (s *GoogleBatchArchivePollerService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	settings := s.compatService.getGoogleBatchArchiveSettings(ctx)
	if settings == nil || !settings.Enabled {
		return
	}
	limit := settings.PollMaxConcurrency * 4
	if limit < 1 {
		limit = 4
	}
	jobs, err := s.jobRepo.ListDueForPoll(ctx, time.Now().UTC(), limit)
	if err != nil || len(jobs) == 0 {
		if err != nil {
			logger.LegacyPrintf("service.google_batch_archive_poller", "[GoogleBatchArchivePoller] list due jobs failed err=%v", err)
		}
		return
	}
	sem := make(chan struct{}, settings.PollMaxConcurrency)
	var wg sync.WaitGroup
	for _, job := range jobs {
		if job == nil {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(item *GoogleBatchArchiveJob) {
			defer wg.Done()
			defer func() { <-sem }()
			s.pollJob(item, settings)
		}(job)
	}
	wg.Wait()
}

func (s *GoogleBatchArchivePollerService) pollJob(job *GoogleBatchArchiveJob, settings *GoogleBatchArchiveSettings) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	account := s.compatService.lookupArchiveExecutionAccountByJob(ctx, job)
	if account == nil {
		s.retryJob(ctx, job, settings, "missing execution account")
		return
	}

	target := googleBatchArchiveTargetForJob(job)
	path := googleBatchArchivePublicBatchPath(job.PublicBatchName)
	if target == googleBatchTargetVertex {
		path = googleBatchArchiveVertexBatchPath(job.ExecutionBatchName)
	}
	if path == "" {
		s.retryJob(ctx, job, settings, "empty poll path")
		return
	}

	input := googleBatchArchiveInputFromJob(job, httpMethodGet, path, "")
	result, err := s.compatService.forwardGoogleBatchToAccount(ctx, input, account, target)
	if err != nil || result == nil {
		s.retryJob(ctx, job, settings, errString(err))
		return
	}
	if result.StatusCode == 404 && job.OfficialExpiresAt != nil && job.OfficialExpiresAt.Before(time.Now().UTC()) {
		job.NextPollAt = nil
		job.PollAttempts++
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
		return
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		s.retryJob(ctx, job, settings, "unexpected status")
		return
	}

	body := result.Body
	if target == googleBatchTargetVertex {
		body = translateVertexBatchPayloadToAIStudio(job, result.Body)
	}
	if err := s.compatService.syncArchiveJobFromBatchPayload(ctx, input, account, job.PublicBatchName, body); err != nil {
		s.retryJob(ctx, job, settings, errString(err))
		return
	}
	updated, err := s.compatService.getGoogleBatchArchiveJobByPublicBatchName(ctx, job.PublicBatchName)
	if err == nil && updated != nil {
		updated.PollAttempts = 0
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, updated)
	}
}

func (s *GoogleBatchArchivePollerService) retryJob(ctx context.Context, job *GoogleBatchArchiveJob, settings *GoogleBatchArchiveSettings, reason string) {
	if job == nil {
		return
	}
	job.PollAttempts++
	next := googleBatchArchiveNextRetryAt(settings, job.PollAttempts)
	job.NextPollAt = &next
	_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
	logger.LegacyPrintf("service.google_batch_archive_poller", "[GoogleBatchArchivePoller] retry job=%s attempts=%d reason=%s next=%s", job.PublicBatchName, job.PollAttempts, reason, next.Format(time.RFC3339))
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

const httpMethodGet = "GET"
