package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type AffiliateRepository interface {
	GetUserAffiliate(ctx context.Context, userID int64) (*UserAffiliate, error)
	EnsureAffiliateRow(ctx context.Context, userID int64, affCode string) (bool, error)
	BindInviterByCode(ctx context.Context, inviteeUserID int64, affCode string) (inviterUserID int64, bound bool, err error)

	AccrueTopupRebate(ctx context.Context, redeemCodeID int64, inviteeUserID int64, creditedAmount float64, policy AffiliateRebatePolicy) (accruedAmount float64, err error)
	ThawFrozenIfNeeded(ctx context.Context, inviterUserID int64) (thawedAmount float64, err error)
	TransferToBalance(ctx context.Context, userID int64) (*AffiliateTransferResult, error)

	ListAffiliateUsers(ctx context.Context, params pagination.PaginationParams, filters AffiliateAdminUserListFilters) ([]AffiliateAdminUser, *pagination.PaginationResult, error)
	LookupAffiliateUsers(ctx context.Context, q string, limit int) ([]AffiliateAdminUser, error)
	UpdateAffiliateUserCustom(ctx context.Context, userID int64, update AffiliateAdminUserCustomUpdate, newAffCodeForClear string) (*UserAffiliate, error)
	ResetAffiliateUserCustom(ctx context.Context, userID int64, newAffCode string) (*UserAffiliate, error)
	BatchUpdateAffiliateUserCustomRates(ctx context.Context, userIDs []int64, customRatePercent float64) (updated int, err error)
}

type AffiliateRebatePolicy struct {
	Enabled              bool
	RebateOnTopupEnabled bool
	DefaultRatePercent   float64
	FreezeHours          int
	DurationDays         int
	PerInviteeCap        float64
}
