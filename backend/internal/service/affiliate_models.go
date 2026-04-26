package service

import "time"

type UserAffiliate struct {
	UserID                  int64
	AffCode                 string
	InviterUserID           *int64
	InviterBoundAt          *time.Time
	InviteeCount            int
	RebateBalance           float64
	RebateFrozenBalance     float64
	LifetimeRebate          float64
	CustomRebateRatePercent *float64
	CustomAffCode           bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type AffiliateUserInfo struct {
	Enabled              bool    `json:"enabled"`
	TransferEnabled      bool    `json:"transfer_enabled"`
	AffCode              string  `json:"aff_code"`
	InviterUserID        *int64  `json:"inviter_user_id,omitempty"`
	InviteeCount         int     `json:"invitee_count"`
	RebateBalance        float64 `json:"rebate_balance"`
	RebateFrozenBalance  float64 `json:"rebate_frozen_balance"`
	LifetimeRebate       float64 `json:"lifetime_rebate"`
	EffectiveRatePercent float64 `json:"effective_rate_percent"`

	RebateOnUsageEnabled bool `json:"rebate_on_usage_enabled"`
	RebateOnTopupEnabled bool `json:"rebate_on_topup_enabled"`

	RebateFreezeHours   int     `json:"rebate_freeze_hours"`
	RebateDurationDays  int     `json:"rebate_duration_days"`
	RebatePerInviteeCap float64 `json:"rebate_per_invitee_cap"`
}

type AffiliateTransferResult struct {
	TransferredAmount float64 `json:"transferred_amount"`
	NewBalance        float64 `json:"new_balance"`
}

type AffiliateAdminUser struct {
	UserID                  int64     `json:"user_id"`
	Email                   string    `json:"email"`
	AffCode                 string    `json:"aff_code"`
	CustomAffCode           bool      `json:"custom_aff_code"`
	CustomRebateRatePercent *float64  `json:"custom_rebate_rate_percent,omitempty"`
	InviterUserID           *int64    `json:"inviter_user_id,omitempty"`
	InviteeCount            int       `json:"invitee_count"`
	RebateBalance           float64   `json:"rebate_balance"`
	RebateFrozenBalance     float64   `json:"rebate_frozen_balance"`
	LifetimeRebate          float64   `json:"lifetime_rebate"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type AffiliateAdminUserListFilters struct {
	HasCustomCode *bool
	HasCustomRate *bool
	HasInviter    *bool
}

type AffiliateAdminUserCustomUpdate struct {
	AffCodeSet bool
	AffCode    *string

	CustomRateSet bool
	CustomRate    *float64
}
