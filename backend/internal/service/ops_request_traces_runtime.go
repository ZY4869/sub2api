package service

import (
	"context"
	"strings"
)

func (s *OpsService) requireRequestTraceEnabled(ctx context.Context) error {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return err
	}
	if !s.getOpsRequestTraceRuntimeConfig(ctx).Enabled {
		return ErrOpsRequestTracesDisabled
	}
	return nil
}

func (s *OpsService) canAccessRequestTraceRaw(ctx context.Context, operatorID int64) bool {
	if operatorID <= 0 {
		return false
	}
	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if strings.TrimSpace(runtimeCfg.EncryptionKey) == "" {
		return false
	}
	if hasOpsRequestTraceAdminRawAccess(ctx) {
		return true
	}
	_, ok := runtimeCfg.RawAccessUserIDs[operatorID]
	return ok
}

func (s *OpsService) requestDetailRetentionDays(ctx context.Context) int {
	days := 30
	if s != nil && s.cfg != nil && s.cfg.Ops.RequestDetails.RetentionDays > 0 {
		days = s.cfg.Ops.RequestDetails.RetentionDays
	}
	if s != nil {
		if advanced, err := s.GetOpsAdvancedSettings(ctx); err == nil && advanced != nil && advanced.RequestDetailRetentionDays > 0 {
			days = advanced.RequestDetailRetentionDays
		}
	}
	return days
}

func (s *OpsService) getOpsRequestTraceRuntimeConfig(ctx context.Context) opsRequestTraceRuntimeConfig {
	cfg := opsRequestTraceRuntimeConfig{
		Enabled:                  true,
		EncryptionKey:            "",
		RawAccessUserIDs:         map[int64]struct{}{},
		RetentionDays:            30,
		PayloadPreviewLimitBytes: opsTracePayloadInlineBytesLimit,
		SuccessSampleRate:        0.1,
		ForceCaptureSlowMs:       opsRequestTraceDefaultSlowMs,
		RawExportMaxRows:         10000,
	}

	if s != nil && s.cfg != nil {
		cfg.Enabled = s.cfg.Ops.RequestDetails.Enabled
		cfg.EncryptionKey = strings.TrimSpace(s.cfg.Ops.RequestDetails.EncryptionKey)
		if s.cfg.Ops.RequestDetails.RetentionDays > 0 {
			cfg.RetentionDays = s.cfg.Ops.RequestDetails.RetentionDays
		}
		if s.cfg.Ops.RequestDetails.SuccessSampleRate >= 0 {
			cfg.SuccessSampleRate = s.cfg.Ops.RequestDetails.SuccessSampleRate
		}
		if s.cfg.Ops.RequestDetails.ForceCaptureSlowMs > 0 {
			cfg.ForceCaptureSlowMs = int64(s.cfg.Ops.RequestDetails.ForceCaptureSlowMs)
		}
		if s.cfg.Ops.RequestDetails.RawExportMaxRows > 0 {
			cfg.RawExportMaxRows = s.cfg.Ops.RequestDetails.RawExportMaxRows
		}
		for _, userID := range s.cfg.Ops.RequestDetails.RawAccessUserIDs {
			if userID > 0 {
				cfg.RawAccessUserIDs[userID] = struct{}{}
			}
		}
	}

	if s != nil {
		if advanced, err := s.GetOpsAdvancedSettings(ctx); err == nil && advanced != nil {
			cfg.Enabled = cfg.Enabled && advanced.RequestDetailsEnabled
			if advanced.RequestDetailRetentionDays > 0 {
				cfg.RetentionDays = advanced.RequestDetailRetentionDays
			}
			if advanced.RequestDetailPayloadPreviewLimitBytes > 0 {
				cfg.PayloadPreviewLimitBytes = advanced.RequestDetailPayloadPreviewLimitBytes
			}
			if advanced.SuccessSampleRate >= 0 {
				cfg.SuccessSampleRate = advanced.SuccessSampleRate
			}
			if advanced.ForceCaptureSlowMs > 0 {
				cfg.ForceCaptureSlowMs = int64(advanced.ForceCaptureSlowMs)
			}
			if advanced.RawExportMaxRows > 0 {
				cfg.RawExportMaxRows = advanced.RawExportMaxRows
			}
		}
	}

	if cfg.SuccessSampleRate < 0 {
		cfg.SuccessSampleRate = 0
	}
	if cfg.SuccessSampleRate > 1 {
		cfg.SuccessSampleRate = 1
	}
	if cfg.PayloadPreviewLimitBytes <= 0 {
		cfg.PayloadPreviewLimitBytes = opsTracePayloadInlineBytesLimit
	}
	if cfg.ForceCaptureSlowMs <= 0 {
		cfg.ForceCaptureSlowMs = opsRequestTraceDefaultSlowMs
	}
	if cfg.RawExportMaxRows <= 0 {
		cfg.RawExportMaxRows = 10000
	}
	return cfg
}
