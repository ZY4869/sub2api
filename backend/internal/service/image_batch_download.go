package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type ImageBatchContent struct {
	ContentType string
	Body        []byte
	FileName    string
}

func (s *ImageBatchService) GetItemContent(ctx context.Context, userID int64, jobID string, customID string) (*ImageBatchContent, error) {
	job, err := s.repo.GetJobForUser(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	release, err := s.acquireImageBatchDownload(ctx, job)
	if err != nil {
		recordImageBatchDownloadFailure()
		return nil, err
	}
	defer release()
	output, err := s.repo.GetOutputForUser(ctx, userID, jobID, customID)
	if err != nil {
		return nil, err
	}
	if output == nil || len(output.Content) == 0 {
		return nil, ErrImageBatchOutputNotFound
	}
	if limit := s.imageBatchDownloadLimitForJob(ctx, job); limit > 0 && output.SizeBytes > limit {
		recordImageBatchDownloadFailure()
		return nil, ErrImageBatchDownloadTooLarge
	}
	return &ImageBatchContent{
		ContentType: firstNonEmptyString(output.ContentType, "image/png"),
		Body:        append([]byte(nil), output.Content...),
		FileName:    imageBatchOutputFileName(output.CustomID, output.ContentType),
	}, nil
}

func (s *ImageBatchService) DownloadZip(ctx context.Context, userID int64, jobID string) (*ImageBatchContent, error) {
	job, err := s.repo.GetJobForUser(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	release, err := s.acquireImageBatchDownload(ctx, job)
	if err != nil {
		recordImageBatchDownloadFailure()
		return nil, err
	}
	defer release()
	limit := s.imageBatchDownloadLimitForJob(ctx, job)
	outputs, err := s.repo.ListOutputsForUser(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, output := range outputs {
		total += output.SizeBytes
		if limit > 0 && total > limit {
			recordImageBatchDownloadFailure()
			return nil, ErrImageBatchDownloadTooLarge
		}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, output := range outputs {
		if len(output.Content) == 0 {
			continue
		}
		name := imageBatchOutputFileName(output.CustomID, output.ContentType)
		writer, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := writer.Write(output.Content); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &ImageBatchContent{
		ContentType: "application/zip",
		Body:        buf.Bytes(),
		FileName:    "image-batch-" + strings.ReplaceAll(job.ID, "-", "") + ".zip",
	}, nil
}

func (s *ImageBatchService) acquireImageBatchDownload(ctx context.Context, job *ImageBatchJob) (func(), error) {
	releases := make([]func(), 0, 2)
	releaseAll := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	globalRelease, err := s.acquireImageBatchDownloadSlot("global", s.globalImageBatchDownloadConcurrency())
	if err != nil {
		return nil, err
	}
	releases = append(releases, globalRelease)
	if job != nil && job.GroupID != nil {
		groupRelease, err := s.acquireImageBatchDownloadSlot("group:"+strconv.FormatInt(*job.GroupID, 10), s.imageBatchGroupDownloadConcurrency(ctx, job))
		if err != nil {
			releaseAll()
			return nil, err
		}
		releases = append(releases, groupRelease)
	}
	return releaseAll, nil
}

func (s *ImageBatchService) acquireImageBatchDownloadSlot(key string, limit int) (func(), error) {
	if limit <= 0 {
		limit = 1
	}
	s.downloadMu.Lock()
	if s.downloadSlots == nil {
		s.downloadSlots = map[string]chan struct{}{}
		s.downloadLimits = map[string]int{}
	}
	slot := s.downloadSlots[key]
	if slot == nil || s.downloadLimits[key] != limit {
		slot = make(chan struct{}, limit)
		s.downloadSlots[key] = slot
		s.downloadLimits[key] = limit
	}
	s.downloadMu.Unlock()

	select {
	case slot <- struct{}{}:
		return func() { <-slot }, nil
	default:
		return nil, ErrImageBatchDownloadBusy
	}
}

func (s *ImageBatchService) globalImageBatchDownloadConcurrency() int {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.DownloadConcurrency > 0 {
		return s.cfg.ImageBatch.DownloadConcurrency
	}
	return 2
}

func (s *ImageBatchService) imageBatchGroupDownloadConcurrency(ctx context.Context, job *ImageBatchJob) int {
	if s != nil && s.repo != nil && job != nil && job.GroupID != nil {
		if settings, err := s.repo.GetGroupSettings(ctx, *job.GroupID); err == nil && settings != nil && settings.DownloadConcurrency > 0 {
			return settings.DownloadConcurrency
		}
	}
	return s.globalImageBatchDownloadConcurrency()
}

func (s *ImageBatchService) imageBatchDownloadLimitForJob(ctx context.Context, job *ImageBatchJob) int64 {
	if s != nil && s.repo != nil && job != nil && job.GroupID != nil {
		if settings, err := s.repo.GetGroupSettings(ctx, *job.GroupID); err == nil {
			return s.imageBatchDownloadLimitBytes(settings)
		}
	}
	return s.imageBatchDownloadLimitBytes(nil)
}

func (s *ImageBatchService) imageBatchDownloadLimitBytes(settings *ImageBatchGroupSettings) int64 {
	if settings != nil && settings.MaxDownloadBytes > 0 {
		return settings.MaxDownloadBytes
	}
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.DownloadMaxBytes > 0 {
		return s.cfg.ImageBatch.DownloadMaxBytes
	}
	return 100 * 1024 * 1024
}

func imageBatchOutputFileName(customID string, contentType string) string {
	ext := ".png"
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	}
	base := strings.TrimSpace(customID)
	if base == "" {
		base = "output"
	}
	base = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, base)
	if filepath.Ext(base) != "" {
		return base
	}
	return fmt.Sprintf("%s%s", base, ext)
}
