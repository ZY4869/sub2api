package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type UserAffiliate struct {
	UserID                  int64      `json:"user_id"`
	AffCode                 string     `json:"aff_code"`
	InviterUserID           *int64     `json:"inviter_user_id,omitempty"`
	InviterBoundAt          *time.Time `json:"inviter_bound_at,omitempty"`
	InviteeCount            int        `json:"invitee_count"`
	RebateBalance           float64    `json:"rebate_balance"`
	RebateFrozenBalance     float64    `json:"rebate_frozen_balance"`
	LifetimeRebate          float64    `json:"lifetime_rebate"`
	CustomRebateRatePercent *float64   `json:"custom_rebate_rate_percent,omitempty"`
	CustomAffCode           bool       `json:"custom_aff_code"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

func UserAffiliateFromService(row *service.UserAffiliate) *UserAffiliate {
	if row == nil {
		return nil
	}
	return &UserAffiliate{
		UserID:                  row.UserID,
		AffCode:                 row.AffCode,
		InviterUserID:           row.InviterUserID,
		InviterBoundAt:          row.InviterBoundAt,
		InviteeCount:            row.InviteeCount,
		RebateBalance:           row.RebateBalance,
		RebateFrozenBalance:     row.RebateFrozenBalance,
		LifetimeRebate:          row.LifetimeRebate,
		CustomRebateRatePercent: row.CustomRebateRatePercent,
		CustomAffCode:           row.CustomAffCode,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}
