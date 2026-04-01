package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type GoogleBatchArchivePrefetchService struct {
	jobRepo        GoogleBatchArchiveJobRepository
	objectRepo     GoogleBatchArchiveObjectRepository
	compatService  *GeminiMessagesCompatService
	settingService *SettingService
	startOnce      sync.Once
	stopOnce       sync.Once
	stopCh         chan struct{}
}

func NewGoogleBatchArchivePrefetchService(
	jobRepo GoogleBatchArchiveJobRepository,
	objectRepo GoogleBatchArchiveObjectRepository,
	compatService *GeminiMessagesCompatService,
	settingService *SettingService,
) *GoogleBatchArchivePrefetchService {
	return &GoogleBatchArchivePrefetchService{
		jobRepo:        jobRepo,
		objectRepo:     objectRepo,
		compatService:  compatService,
		settingService: settingService,
		stopCh:         make(chan struct{}),
	}
}

func (s *GoogleBatchArchivePrefetchService) Start() {
	if s == nil || s.jobRepo == nil || s.objectRepo == nil || s.compatService == nil || s.settingService == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.loop()
		logger.LegacyPrintf("service.google_batch_archive_prefetch", "[GoogleBatchArchivePrefetch] started")
	})
}

func (s *GoogleBatchArchivePrefetchService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		logger.LegacyPrintf("service.google_batch_archive_prefetch", "[GoogleBatchArchivePrefetch] stopped")
	})
}

func (s *GoogleBatchArchivePrefetchService) loop() {
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

func (s *GoogleBatchArchivePrefetchService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	settings := s.compatService.getGoogleBatchArchiveSettings(ctx)
	if settings == nil || !settings.Enabled {
		return
	}
	limit := settings.PollMaxConcurrency * 2
	if limit < 1 {
		limit = 2
	}
	jobs, err := s.jobRepo.ListDueForPrefetch(ctx, time.Now().UTC(), limit)
	if err != nil || len(jobs) == 0 {
		if err != nil {
			logger.LegacyPrintf("service.google_batch_archive_prefetch", "[GoogleBatchArchivePrefetch] list due jobs failed err=%v", err)
		}
		return
	}
	for _, job := range jobs {
		if job == nil {
			continue
		}
		s.prefetchJob(job, settings)
	}
}

func (s *GoogleBatchArchivePrefetchService) prefetchJob(job *GoogleBatchArchiveJob, settings *GoogleBatchArchiveSettings) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(settings.DownloadTimeoutSeconds+30)*time.Second)
	defer cancel()

	sourceAccount, err := s.compatService.accountRepo.GetByID(ctx, job.SourceAccountID)
	if err != nil || sourceAccount == nil || !sourceAccount.IsBatchArchiveEnabled() || !sourceAccount.IsBatchArchiveAutoPrefetchEnabled() {
		return
	}
	resultFileName, ok := metadataString(job.MetadataJSON, "public_result_file_name")
	if !ok {
		return
	}
	object, _ := s.compatService.getGoogleBatchArchiveObject(ctx, GoogleBatchArchiveResourceKindFile, resultFileName)
	if object != nil && strings.TrimSpace(object.RelativePath) != "" {
		job.ArchiveState = GoogleBatchArchiveLifecycleArchived
		job.PrefetchDueAt = nil
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
		return
	}

	account := s.compatService.lookupArchiveExecutionAccountByJob(ctx, job)
	if account == nil {
		return
	}
	if googleBatchArchiveTargetForJob(job) != googleBatchTargetAIStudio {
		return
	}
	input := googleBatchArchiveInputFromJob(job, httpMethodGet, googleBatchArchivePublicFileDownloadPath(resultFileName), "alt=media")
	result, err := s.compatService.forwardGoogleBatchToAccountStream(ctx, input, account, googleBatchTargetAIStudio)
	if err != nil || result == nil || result.StatusCode < 200 || result.StatusCode >= 300 {
		if result != nil && result.Body != nil {
			_ = result.Body.Close()
		}
		next := time.Now().UTC().Add(googleBatchArchivePollInterval(settings))
		job.PrefetchDueAt = &next
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
		return
	}

	if object == nil {
		object = &GoogleBatchArchiveObject{
			JobID:                 job.ID,
			PublicResourceKind:    GoogleBatchArchiveResourceKindFile,
			PublicResourceName:    resultFileName,
			ExecutionResourceName: resultFileName,
			IsResultPayload:       true,
			MetadataJSON:          map[string]any{"public_batch_name": job.PublicBatchName},
		}
	}
	filename := archiveFilenameForPublicResource(resultFileName, googleBatchArchiveResultFilename)
	if err := s.compatService.storeGoogleBatchArchiveObjectReader(ctx, settings, job, object, filename, headerValue(result.Headers, "Content-Type"), result.Body); err != nil {
		_ = result.Body.Close()
		next := time.Now().UTC().Add(googleBatchArchivePollInterval(settings))
		job.PrefetchDueAt = &next
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
		return
	}
	_ = result.Body.Close()
	if err := s.compatService.maybeSettleGoogleBatchArchiveJobFromObject(ctx, input, account, job, settings, object); err != nil {
		next := time.Now().UTC().Add(googleBatchArchivePollInterval(settings))
		job.PrefetchDueAt = &next
		_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
		return
	}
	job.ArchiveState = GoogleBatchArchiveLifecycleArchived
	job.PrefetchDueAt = nil
	_ = s.compatService.upsertGoogleBatchArchiveJob(ctx, job)
	logger.LegacyPrintf("service.google_batch_archive_prefetch", "[GoogleBatchArchivePrefetch] archived job=%s file=%s", job.PublicBatchName, resultFileName)
}
