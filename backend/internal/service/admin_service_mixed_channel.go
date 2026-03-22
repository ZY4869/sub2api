package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *adminServiceImpl) checkMixedChannelRisk(ctx context.Context, currentAccountID int64, currentAccountPlatform string, groupIDs []int64) error {
	currentPlatform := getAccountPlatform(currentAccountPlatform)
	if currentPlatform == "" {
		return nil
	}
	for _, groupID := range groupIDs {
		accounts, err := s.accountRepo.ListByGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("get accounts in group %d: %w", groupID, err)
		}
		for _, account := range accounts {
			if currentAccountID > 0 && account.ID == currentAccountID {
				continue
			}
			otherPlatform := getAccountPlatform(account.Platform)
			if otherPlatform == "" {
				continue
			}
			if currentPlatform != otherPlatform {
				group, _ := s.groupRepo.GetByID(ctx, groupID)
				groupName := fmt.Sprintf("Group %d", groupID)
				if group != nil {
					groupName = group.Name
				}
				return &MixedChannelError{GroupID: groupID, GroupName: groupName, CurrentPlatform: currentPlatform, OtherPlatform: otherPlatform}
			}
		}
	}
	return nil
}

func shouldEnforceMixedChannelCheck(platform string, skipRequested bool) bool {
	if !skipRequested {
		return true
	}
	return platform == PlatformKiro || platform == PlatformCopilot
}

func (s *adminServiceImpl) validateGroupIDsExist(ctx context.Context, groupIDs []int64) error {
	if len(groupIDs) == 0 {
		return nil
	}
	if s.groupRepo == nil {
		return errors.New("group repository not configured")
	}
	if batchReader, ok := s.groupRepo.(groupExistenceBatchReader); ok {
		existsByID, err := batchReader.ExistsByIDs(ctx, groupIDs)
		if err != nil {
			return fmt.Errorf("check groups exists: %w", err)
		}
		for _, groupID := range groupIDs {
			if groupID <= 0 || !existsByID[groupID] {
				return fmt.Errorf("get group: %w", ErrGroupNotFound)
			}
		}
		return nil
	}
	for _, groupID := range groupIDs {
		if _, err := s.groupRepo.GetByID(ctx, groupID); err != nil {
			return fmt.Errorf("get group: %w", err)
		}
	}
	return nil
}
func (s *adminServiceImpl) CheckMixedChannelRisk(ctx context.Context, currentAccountID int64, currentAccountPlatform string, groupIDs []int64) error {
	return s.checkMixedChannelRisk(ctx, currentAccountID, currentAccountPlatform, groupIDs)
}
func (s *adminServiceImpl) saveProxyLatency(ctx context.Context, proxyID int64, info *ProxyLatencyInfo) {
	if s.proxyLatencyCache == nil || info == nil {
		return
	}
	merged := *info
	if latencies, err := s.proxyLatencyCache.GetProxyLatencies(ctx, []int64{proxyID}); err == nil {
		if existing := latencies[proxyID]; existing != nil {
			if merged.QualityCheckedAt == nil && merged.QualityScore == nil && merged.QualityGrade == "" && merged.QualityStatus == "" && merged.QualitySummary == "" && merged.QualityCFRay == "" {
				merged.QualityStatus = existing.QualityStatus
				merged.QualityScore = existing.QualityScore
				merged.QualityGrade = existing.QualityGrade
				merged.QualitySummary = existing.QualitySummary
				merged.QualityCheckedAt = existing.QualityCheckedAt
				merged.QualityCFRay = existing.QualityCFRay
			}
		}
	}
	if err := s.proxyLatencyCache.SetProxyLatency(ctx, proxyID, &merged); err != nil {
		logger.LegacyPrintf("service.admin", "Warning: store proxy latency cache failed: %v", err)
	}
}
func getAccountPlatform(accountPlatform string) string {
	return DisplayPlatformName(accountPlatform)
}

type MixedChannelError struct {
	GroupID         int64
	GroupName       string
	CurrentPlatform string
	OtherPlatform   string
}

func (e *MixedChannelError) Error() string {
	return fmt.Sprintf("mixed_channel_warning: Group '%s' contains both %s and %s accounts. Using mixed channels in the same context may cause thinking block signature validation issues, which will fallback to non-thinking mode for historical messages.", e.GroupName, e.CurrentPlatform, e.OtherPlatform)
}
