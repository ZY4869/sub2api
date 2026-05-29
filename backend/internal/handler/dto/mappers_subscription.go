package dto

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func UserSubscriptionFromService(sub *service.UserSubscription) *UserSubscription {
	if sub == nil {
		return nil
	}
	out := userSubscriptionFromServiceBase(sub)
	return &out
}

// UserSubscriptionFromServiceAdmin converts a service UserSubscription to DTO for admin users.
// It includes assignment metadata and notes.
func UserSubscriptionFromServiceAdmin(sub *service.UserSubscription) *AdminUserSubscription {
	if sub == nil {
		return nil
	}
	return &AdminUserSubscription{
		UserSubscription: userSubscriptionFromServiceBase(sub),
		AssignedBy:       sub.AssignedBy,
		AssignedAt:       sub.AssignedAt,
		Notes:            sub.Notes,
		AssignedByUser:   UserFromServiceShallow(sub.AssignedByUser),
	}
}

func userSubscriptionFromServiceBase(sub *service.UserSubscription) UserSubscription {
	return UserSubscription{
		ID:                     sub.ID,
		UserID:                 sub.UserID,
		GroupID:                sub.GroupID,
		StartsAt:               sub.StartsAt,
		ExpiresAt:              sub.ExpiresAt,
		Status:                 sub.Status,
		DailyWindowStart:       sub.DailyWindowStart,
		WeeklyWindowStart:      sub.WeeklyWindowStart,
		MonthlyWindowStart:     sub.MonthlyWindowStart,
		DailyUsageUSD:          sub.DailyUsageUSD,
		WeeklyUsageUSD:         sub.WeeklyUsageUSD,
		MonthlyUsageUSD:        sub.MonthlyUsageUSD,
		DailyUsageByCurrency:   cloneUsageCostByCurrency(sub.DailyUsageByCurrency),
		WeeklyUsageByCurrency:  cloneUsageCostByCurrency(sub.WeeklyUsageByCurrency),
		MonthlyUsageByCurrency: cloneUsageCostByCurrency(sub.MonthlyUsageByCurrency),
		CreatedAt:              sub.CreatedAt,
		UpdatedAt:              sub.UpdatedAt,
		User:                   UserFromServiceShallow(sub.User),
		Group:                  GroupFromServiceShallow(sub.Group),
	}
}

func BulkAssignResultFromService(r *service.BulkAssignResult) *BulkAssignResult {
	if r == nil {
		return nil
	}
	subs := make([]AdminUserSubscription, 0, len(r.Subscriptions))
	for i := range r.Subscriptions {
		subs = append(subs, *UserSubscriptionFromServiceAdmin(&r.Subscriptions[i]))
	}
	statuses := make(map[string]string, len(r.Statuses))
	for userID, status := range r.Statuses {
		statuses[strconv.FormatInt(userID, 10)] = status
	}
	return &BulkAssignResult{
		SuccessCount:  r.SuccessCount,
		CreatedCount:  r.CreatedCount,
		ReusedCount:   r.ReusedCount,
		FailedCount:   r.FailedCount,
		Subscriptions: subs,
		Errors:        r.Errors,
		Statuses:      statuses,
	}
}
