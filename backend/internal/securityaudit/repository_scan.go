package securityaudit

import (
	"database/sql"
	"encoding/json"
)

type rowScanner interface{ Scan(dest ...any) error }

func scanEvent(scanner rowScanner, includeFullText bool) (*Event, error) {
	event := &Event{}
	var userID, apiKeyID, groupID sql.NullInt64
	var categories, matched, scores, evidence []byte
	err := scanner.Scan(&event.ID, &event.JobID, &event.RequestID, &userID, &event.Username,
		&event.UserEmail, &apiKeyID, &event.APIKeyName, &groupID, &event.GroupName,
		&event.Provider, &event.Endpoint, &event.Protocol, &event.Model, &event.PromptHash,
		&event.RedactedPreview, &event.FullPrompt, &event.Stage, &event.Decision, &event.RiskLevel,
		&event.Action, &categories, &matched, &scores, &evidence, &event.ScannerBackend,
		&event.ScannerVersion, &event.GuardEndpointID, &event.PolicyID, &event.PolicyVersion,
		&event.ConfigVersion, &event.ChunkTotal, &event.LatencyMS, &event.CreatedAt)
	if err != nil {
		return nil, err
	}
	event.UserID, event.APIKeyID, event.GroupID = nullInt64Ptr(userID), nullInt64Ptr(apiKeyID), nullInt64Ptr(groupID)
	event.Categories, event.MatchedScanners = jsonStrings(categories), jsonStrings(matched)
	event.ScannerScores, event.ScannerEvidence = jsonFloatMap(scores), jsonStringMap(evidence)
	event.IssueSummaries = BuildIssueSummaries(NormalizedResult{Categories: event.Categories, ScannerEvidence: event.ScannerEvidence})
	if !includeFullText {
		event.FullPrompt = ""
	}
	return event, nil
}

func scanJob(scanner rowScanner) (*Job, error) {
	job := &Job{}
	var userID, apiKeyID, groupID sql.NullInt64
	var processingStarted, processedAt sql.NullTime
	err := scanner.Scan(&job.ID, &job.RequestID, &userID, &job.Username, &job.UserEmail,
		&apiKeyID, &job.APIKeyName, &groupID, &job.GroupName, &job.Provider, &job.Endpoint,
		&job.Protocol, &job.Model, &job.PromptHash, &job.RedactedPreview, &job.PromptLength,
		&job.MessageCount, &job.Stage, &job.ExecutionMode, &job.ConfigVersion, &job.Status,
		&job.Attempts, &job.MaxAttempts, &job.NextAttemptAt, &processingStarted, &processedAt,
		&job.LastErrorCode, &job.LastErrorMessage, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return nil, err
	}
	job.UserID, job.APIKeyID, job.GroupID = nullInt64Ptr(userID), nullInt64Ptr(apiKeyID), nullInt64Ptr(groupID)
	if processingStarted.Valid {
		job.ProcessingStarted = &processingStarted.Time
	}
	if processedAt.Valid {
		job.ProcessedAt = &processedAt.Time
	}
	return job, nil
}

func scanQueueStat(scanner rowScanner, stats *QueueStats) error {
	var status string
	var count int64
	if err := scanner.Scan(&status, &count); err != nil {
		return err
	}
	switch status {
	case JobStatusQueued:
		stats.Queued = count
	case JobStatusProcessing:
		stats.Processing = count
	case JobStatusRetry:
		stats.Retry = count
	case JobStatusDone:
		stats.Done = count
	case JobStatusFailed:
		stats.Failed = count
	}
	return nil
}

func scanDeleteResult(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) (*DeleteResult, error) {
	result := &DeleteResult{}
	for rows.Next() {
		var jobID int64
		if err := rows.Scan(&jobID); err != nil {
			return nil, err
		}
		result.Deleted++
		result.JobIDs = append(result.JobIDs, jobID)
	}
	return result, rows.Err()
}

func jsonStrings(raw []byte) []string {
	var values []string
	_ = json.Unmarshal(raw, &values)
	return values
}

func jsonFloatMap(raw []byte) map[string]float64 {
	out := map[string]float64{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func jsonStringMap(raw []byte) map[string]string {
	out := map[string]string{}
	_ = json.Unmarshal(raw, &out)
	return out
}
