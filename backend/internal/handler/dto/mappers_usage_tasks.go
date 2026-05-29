package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

func UsageCleanupTaskFromService(task *service.UsageCleanupTask) *UsageCleanupTask {
	if task == nil {
		return nil
	}
	return &UsageCleanupTask{
		ID:     task.ID,
		Status: task.Status,
		Filters: UsageCleanupFilters{
			StartTime:   task.Filters.StartTime,
			EndTime:     task.Filters.EndTime,
			UserID:      task.Filters.UserID,
			APIKeyID:    task.Filters.APIKeyID,
			AccountID:   task.Filters.AccountID,
			GroupID:     task.Filters.GroupID,
			Model:       task.Filters.Model,
			RequestType: requestTypeStringPtr(task.Filters.RequestType),
			Stream:      task.Filters.Stream,
			BillingType: task.Filters.BillingType,
		},
		CreatedBy:    task.CreatedBy,
		DeletedRows:  task.DeletedRows,
		ErrorMessage: task.ErrorMsg,
		CanceledBy:   task.CanceledBy,
		CanceledAt:   task.CanceledAt,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}

func UsageRepairTaskFromService(task *service.UsageRepairTask) *UsageRepairTask {
	if task == nil {
		return nil
	}
	return &UsageRepairTask{
		ID:            task.ID,
		Kind:          task.Kind,
		Days:          task.Days,
		Status:        task.Status,
		CreatedBy:     task.CreatedBy,
		ProcessedRows: task.ProcessedRows,
		RepairedRows:  task.RepairedRows,
		SkippedRows:   task.SkippedRows,
		ErrorMessage:  task.ErrorMsg,
		CanceledBy:    task.CanceledBy,
		CanceledAt:    task.CanceledAt,
		StartedAt:     task.StartedAt,
		FinishedAt:    task.FinishedAt,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}
