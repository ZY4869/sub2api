package service

import (
	"context"
	"encoding/json"
	"time"
)

func (s *OpsService) insertRequestTraceAudit(ctx context.Context, traceID *int64, operatorID int64, action OpsRequestTraceAuditAction, meta map[string]any) error {
	if s == nil || s.opsRepo == nil || operatorID <= 0 {
		return nil
	}
	var metaJSON *string
	if len(meta) > 0 {
		if raw, err := json.Marshal(meta); err == nil {
			value := string(raw)
			metaJSON = &value
		}
	}
	return s.opsRepo.InsertRequestTraceAudit(ctx, &OpsInsertRequestTraceAuditInput{
		TraceID:    traceID,
		OperatorID: operatorID,
		Action:     action,
		MetaJSON:   metaJSON,
		CreatedAt:  time.Now().UTC(),
	})
}
