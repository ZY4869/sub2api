package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func RedeemCodeFromService(rc *service.RedeemCode) *RedeemCode {
	if rc == nil {
		return nil
	}
	out := redeemCodeFromServiceBase(rc)
	return &out
}

// RedeemCodeFromServiceAdmin converts a service RedeemCode to DTO for admin users.
// It includes notes - user-facing endpoints must not use this.
func RedeemCodeFromServiceAdmin(rc *service.RedeemCode) *AdminRedeemCode {
	if rc == nil {
		return nil
	}
	return &AdminRedeemCode{
		RedeemCode: redeemCodeFromServiceBase(rc),
		Notes:      rc.Notes,
	}
}

func redeemCodeFromServiceBase(rc *service.RedeemCode) RedeemCode {
	status := rc.Status
	if rc.IsExpired(time.Now()) {
		status = service.StatusExpired
	}
	out := RedeemCode{
		ID:           rc.ID,
		Code:         rc.Code,
		Type:         rc.Type,
		Value:        rc.Value,
		Status:       status,
		UsedBy:       rc.UsedBy,
		UsedAt:       rc.UsedAt,
		CreatedAt:    rc.CreatedAt,
		ExpiresAt:    rc.ExpiresAt,
		GroupID:      rc.GroupID,
		ValidityDays: rc.ValidityDays,
		User:         UserFromServiceShallow(rc.User),
		Group:        GroupFromServiceShallow(rc.Group),
	}

	// For admin_balance/admin_concurrency types, include notes so users can see
	// why they were charged or credited by admin.
	if (rc.Type == "admin_balance" || rc.Type == "admin_concurrency") && rc.Notes != "" {
		out.Notes = &rc.Notes
	}

	return out
}

func PromoCodeFromService(pc *service.PromoCode) *PromoCode {
	if pc == nil {
		return nil
	}
	return &PromoCode{
		ID:          pc.ID,
		Code:        pc.Code,
		BonusAmount: pc.BonusAmount,
		MaxUses:     pc.MaxUses,
		UsedCount:   pc.UsedCount,
		Status:      pc.Status,
		ExpiresAt:   pc.ExpiresAt,
		Notes:       pc.Notes,
		CreatedAt:   pc.CreatedAt,
		UpdatedAt:   pc.UpdatedAt,
	}
}

func PromoCodeUsageFromService(u *service.PromoCodeUsage) *PromoCodeUsage {
	if u == nil {
		return nil
	}
	return &PromoCodeUsage{
		ID:          u.ID,
		PromoCodeID: u.PromoCodeID,
		UserID:      u.UserID,
		BonusAmount: u.BonusAmount,
		UsedAt:      u.UsedAt,
		User:        UserFromServiceShallow(u.User),
	}
}
