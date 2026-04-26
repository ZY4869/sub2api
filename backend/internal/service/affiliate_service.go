package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/lib/pq"
)

var (
	ErrAffiliateTransferDisabled = infraerrors.Forbidden("AFFILIATE_TRANSFER_DISABLED", "affiliate transfer is disabled")
	ErrAffiliateCodeConflict     = infraerrors.Conflict("AFFILIATE_CODE_CONFLICT", "affiliate code already exists")
)

type AffiliateService struct {
	settingService *SettingService
	repo           AffiliateRepository
}

func NewAffiliateService(settingService *SettingService, repo AffiliateRepository) *AffiliateService {
	return &AffiliateService{settingService: settingService, repo: repo}
}

func normalizeAffiliateCode(input string) string {
	input = strings.TrimSpace(strings.ToUpper(input))
	input = strings.ReplaceAll(input, "-", "")
	input = strings.ReplaceAll(input, " ", "")
	return input
}

func isAffiliateCodeLike(input string) bool {
	if input == "" {
		return false
	}
	for _, ch := range input {
		if ch >= 'A' && ch <= 'Z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func generateAffiliateCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}
	if length > 64 {
		return "", fmt.Errorf("length too large")
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	out := make([]byte, length)
	for i := range buf {
		out[i] = charset[int(buf[i])%len(charset)]
	}
	return string(out), nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	return false
}

func (s *AffiliateService) EnsureAffiliateRow(ctx context.Context, userID int64) (*UserAffiliate, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("affiliate service repo is nil")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}

	codeLength := 10
	if s.settingService != nil {
		all, err := s.settingService.GetAllSettings(ctx)
		if err == nil && all != nil && all.AffiliateAffCodeLength > 0 {
			codeLength = all.AffiliateAffCodeLength
		}
	}
	if codeLength < 6 {
		codeLength = 6
	}
	if codeLength > 32 {
		codeLength = 32
	}

	// Best-effort insert with random code, retrying on code collisions.
	for attempt := 0; attempt < 10; attempt++ {
		code, err := generateAffiliateCode(codeLength)
		if err != nil {
			return nil, err
		}
		_, err = s.repo.EnsureAffiliateRow(ctx, userID, code)
		if err == nil {
			return s.repo.GetUserAffiliate(ctx, userID)
		}
		if isUniqueViolation(err) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("failed to generate unique affiliate code after retries")
}

func (s *AffiliateService) BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) {
	if s == nil || s.repo == nil {
		return
	}
	if inviteeUserID <= 0 {
		return
	}
	affCode = normalizeAffiliateCode(affCode)
	if !isAffiliateCodeLike(affCode) {
		return
	}

	if _, err := s.EnsureAffiliateRow(ctx, inviteeUserID); err != nil {
		slog.Warn("affiliate: ensure invitee row failed", "user_id", inviteeUserID, "error", err)
		return
	}

	inviterUserID, bound, err := s.repo.BindInviterByCode(ctx, inviteeUserID, affCode)
	if err != nil {
		slog.Warn("affiliate: bind inviter failed", "user_id", inviteeUserID, "aff_code", affCode, "error", err)
		return
	}
	if !bound {
		return
	}
	slog.Info("affiliate: inviter bound", "user_id", inviteeUserID, "inviter_user_id", inviterUserID)
}

func (s *AffiliateService) GetMyAffiliateInfo(ctx context.Context, userID int64) (*AffiliateUserInfo, error) {
	row, err := s.EnsureAffiliateRow(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings := &SystemSettings{
		AffiliateEnabled:              false,
		AffiliateTransferEnabled:      true,
		AffiliateRebateOnUsageEnabled: true,
		AffiliateRebateOnTopupEnabled: true,
		AffiliateRebateRate:           20,
		AffiliateRebateFreezeHours:    0,
		AffiliateRebateDurationDays:   0,
		AffiliateRebatePerInviteeCap:  0,
	}
	if s.settingService != nil {
		if all, err := s.settingService.GetAllSettings(ctx); err == nil && all != nil {
			settings = all
		}
	}

	if row.RebateFrozenBalance > 0 {
		if _, err := s.repo.ThawFrozenIfNeeded(ctx, userID); err != nil {
			slog.Warn("affiliate: thaw failed", "user_id", userID, "error", err)
		} else {
			row, _ = s.repo.GetUserAffiliate(ctx, userID)
		}
	}

	effectiveRate := settings.AffiliateRebateRate
	if row.CustomRebateRatePercent != nil {
		effectiveRate = *row.CustomRebateRatePercent
	}

	return &AffiliateUserInfo{
		Enabled:              settings.AffiliateEnabled,
		TransferEnabled:      settings.AffiliateTransferEnabled,
		AffCode:              row.AffCode,
		InviterUserID:        row.InviterUserID,
		InviteeCount:         row.InviteeCount,
		RebateBalance:        row.RebateBalance,
		RebateFrozenBalance:  row.RebateFrozenBalance,
		LifetimeRebate:       row.LifetimeRebate,
		EffectiveRatePercent: effectiveRate,

		RebateOnUsageEnabled: settings.AffiliateRebateOnUsageEnabled,
		RebateOnTopupEnabled: settings.AffiliateRebateOnTopupEnabled,
		RebateFreezeHours:    settings.AffiliateRebateFreezeHours,
		RebateDurationDays:   settings.AffiliateRebateDurationDays,
		RebatePerInviteeCap:  settings.AffiliateRebatePerInviteeCap,
	}, nil
}

func (s *AffiliateService) TransferToBalance(ctx context.Context, userID int64) (*AffiliateTransferResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("affiliate service repo is nil")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}

	settings := &SystemSettings{AffiliateTransferEnabled: true}
	if s.settingService != nil {
		if all, err := s.settingService.GetAllSettings(ctx); err == nil && all != nil {
			settings = all
		}
	}
	if !settings.AffiliateTransferEnabled {
		return nil, ErrAffiliateTransferDisabled
	}

	if _, err := s.EnsureAffiliateRow(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.TransferToBalance(ctx, userID)
}

func (s *AffiliateService) AccrueTopupRebateBestEffort(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64) {
	if s == nil || s.repo == nil {
		return
	}
	if creditedAmount <= 0 || redeemCodeID <= 0 || inviteeUserID <= 0 {
		return
	}

	settings := &SystemSettings{}
	if s.settingService != nil {
		all, err := s.settingService.GetAllSettings(ctx)
		if err != nil {
			slog.Warn("affiliate: load settings failed, skip topup accrual", "error", err)
			return
		}
		settings = all
	}

	policy := AffiliateRebatePolicy{
		Enabled:              settings.AffiliateEnabled,
		RebateOnTopupEnabled: settings.AffiliateRebateOnTopupEnabled,
		DefaultRatePercent:   settings.AffiliateRebateRate,
		FreezeHours:          settings.AffiliateRebateFreezeHours,
		DurationDays:         settings.AffiliateRebateDurationDays,
		PerInviteeCap:        settings.AffiliateRebatePerInviteeCap,
	}
	if !policy.Enabled || !policy.RebateOnTopupEnabled {
		return
	}

	if _, err := s.EnsureAffiliateRow(ctx, inviteeUserID); err != nil {
		slog.Warn("affiliate: ensure invitee row failed, skip topup accrual", "user_id", inviteeUserID, "error", err)
		return
	}

	accrued, err := s.repo.AccrueTopupRebate(ctx, redeemCodeID, inviteeUserID, creditedAmount, policy)
	if err != nil {
		slog.Warn("affiliate: topup accrue failed", "redeem_code_id", redeemCodeID, "user_id", inviteeUserID, "error", err)
		return
	}
	if accrued <= 0 {
		return
	}
	slog.Info("affiliate: topup accrued", "redeem_code_id", redeemCodeID, "user_id", inviteeUserID, "amount", accrued)
}

func (s *AffiliateService) ListAdminUsers(ctx context.Context, params pagination.PaginationParams, filters AffiliateAdminUserListFilters) ([]AffiliateAdminUser, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, errors.New("affiliate service repo is nil")
	}
	return s.repo.ListAffiliateUsers(ctx, params, filters)
}

func (s *AffiliateService) LookupAdminUsers(ctx context.Context, q string, limit int) ([]AffiliateAdminUser, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("affiliate service repo is nil")
	}
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return s.repo.LookupAffiliateUsers(ctx, q, limit)
}

func (s *AffiliateService) UpdateAdminUserCustom(ctx context.Context, userID int64, update AffiliateAdminUserCustomUpdate) (*UserAffiliate, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("affiliate service repo is nil")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}

	if _, err := s.EnsureAffiliateRow(ctx, userID); err != nil {
		return nil, err
	}

	customCodeRequested := update.AffCodeSet && update.AffCode != nil && strings.TrimSpace(*update.AffCode) != ""
	clearCodeRequested := update.AffCodeSet && !customCodeRequested

	// Normalize provided code early (if any).
	if customCodeRequested {
		code := normalizeAffiliateCode(*update.AffCode)
		update.AffCode = &code
	}

	codeLength := 10
	if s.settingService != nil {
		all, err := s.settingService.GetAllSettings(ctx)
		if err == nil && all != nil && all.AffiliateAffCodeLength > 0 {
			codeLength = all.AffiliateAffCodeLength
		}
	}
	if codeLength < 6 {
		codeLength = 6
	}
	if codeLength > 32 {
		codeLength = 32
	}

	// Retry when clearing code because generated codes might (rarely) conflict.
	for attempt := 0; attempt < 10; attempt++ {
		newCodeForClear := ""
		if clearCodeRequested {
			code, err := generateAffiliateCode(codeLength)
			if err != nil {
				return nil, err
			}
			newCodeForClear = code
		}

		row, err := s.repo.UpdateAffiliateUserCustom(ctx, userID, update, newCodeForClear)
		if err == nil {
			return row, nil
		}
		if isUniqueViolation(err) {
			if customCodeRequested {
				return nil, ErrAffiliateCodeConflict.WithCause(err)
			}
			if clearCodeRequested {
				continue
			}
		}
		return nil, err
	}
	return nil, fmt.Errorf("failed to generate unique affiliate code after retries")
}

func (s *AffiliateService) ResetAdminUserCustom(ctx context.Context, userID int64) (*UserAffiliate, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("affiliate service repo is nil")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("user_id must be positive")
	}

	if _, err := s.EnsureAffiliateRow(ctx, userID); err != nil {
		return nil, err
	}

	codeLength := 10
	if s.settingService != nil {
		all, err := s.settingService.GetAllSettings(ctx)
		if err == nil && all != nil && all.AffiliateAffCodeLength > 0 {
			codeLength = all.AffiliateAffCodeLength
		}
	}
	if codeLength < 6 {
		codeLength = 6
	}
	if codeLength > 32 {
		codeLength = 32
	}

	for attempt := 0; attempt < 10; attempt++ {
		code, err := generateAffiliateCode(codeLength)
		if err != nil {
			return nil, err
		}
		row, err := s.repo.ResetAffiliateUserCustom(ctx, userID, code)
		if err == nil {
			return row, nil
		}
		if isUniqueViolation(err) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("failed to generate unique affiliate code after retries")
}

func (s *AffiliateService) BatchUpdateAdminUserRates(ctx context.Context, userIDs []int64, customRatePercent float64) (int, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("affiliate service repo is nil")
	}
	if len(userIDs) == 0 {
		return 0, nil
	}
	if customRatePercent < 0 {
		customRatePercent = 0
	}
	if customRatePercent > 100 {
		customRatePercent = 100
	}

	for _, id := range userIDs {
		if id <= 0 {
			return 0, fmt.Errorf("user_id must be positive")
		}
	}
	for _, id := range userIDs {
		if _, err := s.EnsureAffiliateRow(ctx, id); err != nil {
			slog.Warn("affiliate: ensure row failed in batch", "user_id", id, "error", err)
		}
	}
	updated, err := s.repo.BatchUpdateAffiliateUserCustomRates(ctx, userIDs, customRatePercent)
	if err != nil {
		return 0, err
	}
	return updated, nil
}
