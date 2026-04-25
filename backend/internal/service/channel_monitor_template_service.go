package service

import (
	"context"
	"errors"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ChannelMonitorTemplateService struct {
	tplRepo     ChannelMonitorTemplateRepository
	monitorRepo ChannelMonitorRepository
}

func NewChannelMonitorTemplateService(tplRepo ChannelMonitorTemplateRepository, monitorRepo ChannelMonitorRepository) *ChannelMonitorTemplateService {
	return &ChannelMonitorTemplateService{
		tplRepo:     tplRepo,
		monitorRepo: monitorRepo,
	}
}

func (s *ChannelMonitorTemplateService) ListAll(ctx context.Context) ([]*ChannelMonitorRequestTemplate, error) {
	return s.tplRepo.ListAll(ctx)
}

func (s *ChannelMonitorTemplateService) GetByID(ctx context.Context, id int64) (*ChannelMonitorRequestTemplate, error) {
	return s.tplRepo.GetByID(ctx, id)
}

func (s *ChannelMonitorTemplateService) Create(ctx context.Context, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error) {
	normalized, err := normalizeChannelMonitorTemplate(tpl)
	if err != nil {
		return nil, err
	}
	return s.tplRepo.Create(ctx, normalized)
}

func (s *ChannelMonitorTemplateService) Update(ctx context.Context, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error) {
	if tpl == nil {
		return nil, errors.New("nil template")
	}
	// ensure exists
	if _, err := s.tplRepo.GetByID(ctx, tpl.ID); err != nil {
		return nil, err
	}

	normalized, err := normalizeChannelMonitorTemplate(tpl)
	if err != nil {
		return nil, err
	}
	normalized.ID = tpl.ID
	return s.tplRepo.Update(ctx, normalized)
}

func (s *ChannelMonitorTemplateService) Delete(ctx context.Context, id int64) error {
	return s.tplRepo.Delete(ctx, id)
}

func (s *ChannelMonitorTemplateService) ListAssociatedMonitors(ctx context.Context, templateID int64) ([]*ChannelMonitor, error) {
	return s.tplRepo.ListAssociatedMonitors(ctx, templateID)
}

func (s *ChannelMonitorTemplateService) ApplyToMonitor(ctx context.Context, templateID int64, monitorID int64) (*ChannelMonitor, error) {
	tpl, err := s.tplRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(strings.ToLower(monitor.Provider)) != strings.TrimSpace(strings.ToLower(tpl.Provider)) {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_TEMPLATE_PROVIDER_MISMATCH", "template provider does not match monitor provider")
	}

	monitor.TemplateID = &tpl.ID
	monitor.ExtraHeaders = tpl.ExtraHeaders
	monitor.BodyOverrideMode = tpl.BodyOverrideMode
	monitor.BodyOverride = tpl.BodyOverride
	return s.monitorRepo.Update(ctx, monitor)
}

func normalizeChannelMonitorTemplate(tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error) {
	if tpl == nil {
		return nil, errors.New("nil template")
	}
	out := *tpl
	out.Name = strings.TrimSpace(out.Name)
	out.Provider = strings.TrimSpace(strings.ToLower(out.Provider))
	out.BodyOverrideMode = strings.TrimSpace(strings.ToLower(out.BodyOverrideMode))

	if out.Name == "" || len(out.Name) > 100 {
		return nil, infraerrors.BadRequest("CHANNEL_MONITOR_TEMPLATE_NAME_INVALID", "invalid name")
	}
	if !isValidChannelMonitorProvider(out.Provider) {
		return nil, ErrChannelMonitorInvalidProvider
	}
	if out.BodyOverrideMode == "" {
		out.BodyOverrideMode = ChannelMonitorBodyOverrideModeOff
	}
	if !isValidChannelMonitorBodyOverrideMode(out.BodyOverrideMode) {
		return nil, ErrChannelMonitorInvalidOverrideMode
	}

	out.ExtraHeaders = normalizeChannelMonitorHeaders(out.ExtraHeaders)
	out.BodyOverride = ensureAnyMap(out.BodyOverride)
	if out.BodyOverrideMode == ChannelMonitorBodyOverrideModeReplace && len(out.BodyOverride) == 0 {
		return nil, ErrChannelMonitorInvalidBodyOverride
	}

	if out.Description != nil {
		v := strings.TrimSpace(*out.Description)
		if v == "" {
			out.Description = nil
		} else {
			out.Description = &v
		}
	}
	return &out, nil
}
