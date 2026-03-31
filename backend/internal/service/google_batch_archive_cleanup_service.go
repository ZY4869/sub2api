package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type GoogleBatchArchiveCleanupService struct {
	jobRepo         GoogleBatchArchiveJobRepository
	objectRepo      GoogleBatchArchiveObjectRepository
	compatService   *GeminiMessagesCompatService
	settingService  *SettingService
	startOnce       sync.Once
	stopOnce        sync.Once
	stopCh          chan struct{}
}

func NewGoogleBatchArchiveCleanupService(
	jobRepo GoogleBatchArchiveJobRepository,
	objectRepo GoogleBatchArchiveObjectRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchiveCleanupService {
	return &GoogleBatchArchiveCleanupService{
		jobRepo:        jobRepo,
		objectRepo:     objectRepo,
		compatService:  compatService,
		settingService: settingService,
		stopCh:         make(chan struct{}),
	}
}

func (s *GoogleBatchArchiveCleanupService) Start() {
	if s == nil || s.jobRepo == nil || s.objectRepo == nil || s.compatService == nil || s.settingService == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.loop()
		logger.LegacyPrintf("service.google_batch_archive_cleanup", "[GoogleBatchArchiveCleanup] started")
	})
}

func (s *GoogleBatchArchiveCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		logger.LegacyPrintf("service.google_batch_archive_cleanup", "[GoogleBatchArchiveCleanup] stopped")
	})
}

func (s *GoogleBatchArchiveCleanupService) loop() {
	for {
		s.runOnce()
		interval := googleBatchArchiveCleanupInterval(s.compatService.getGoogleBatchArchiveSettings(context.Background()))
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

func (s *GoogleBatchArchiveCleanupService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	settings := s.compatService.getGoogleBatchArchiveSettings(ctx)
	if settings == nil || !settings.Enabled {
		return
	}
	jobs, err := s.jobRepo.ListExpiredForCleanup(ctx, time.Now().UTC(), 100)
	if err != nil || len(jobs) == 0 {
		if err != nil {
			logger.LegacyPrintf("service.google_batch_archive_cleanup", "[GoogleBatchArchiveCleanup] list expired jobs failed err=%v", err)
		}
		return
	}
	for _, job := range jobs {
		if job == nil {
			continue
		}
		s.cleanupJob(job, settings)
	}
}

func (s *GoogleBatchArchiveCleanupService) cleanupJob(job *GoogleBatchArchiveJob, settings *GoogleBatchArchiveSettings) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	if s.compatService.googleBatchArchiveStorage != nil {
		_ = s.compatService.googleBatchArchiveStorage.DeleteJobDir(settings, job)
	}
	_ = s.objectRepo.SoftDeleteByJobID(ctx, job.ID)
	if strings.TrimSpace(job.PublicBatchName) != "" {
		s.compatService.releaseGoogleBatchQuota(ctx, job.PublicBatchName, GoogleBatchQuotaReservationStatusReleased)
	}
	if executionName := strings.TrimSpace(job.ExecutionBatchName); executionName != "" && executionName != strings.TrimSpace(job.PublicBatchName) {
		s.compatService.releaseGoogleBatchQuota(ctx, executionName, GoogleBatchQuotaReservationStatusReleased)
	}
	_ = s.jobRepo.SoftDelete(ctx, job.ID)
	logger.LegacyPrintf("service.google_batch_archive_cleanup", "[GoogleBatchArchiveCleanup] deleted local archive for job=%s", job.PublicBatchName)
}
