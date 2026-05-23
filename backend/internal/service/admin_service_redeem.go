package service

import (
	"context"
	"errors"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type redeemCodeSortedLister interface {
	ListWithFiltersAndSort(ctx context.Context, params pagination.PaginationParams, codeType, status, search, sortBy, sortOrder string) ([]RedeemCode, *pagination.PaginationResult, error)
}

func (s *adminServiceImpl) ListRedeemCodes(ctx context.Context, page, pageSize int, codeType, status, search string) ([]RedeemCode, int64, error) {
	return s.ListRedeemCodesWithOptions(ctx, RedeemCodeListInput{
		Page:     page,
		PageSize: pageSize,
		Type:     codeType,
		Status:   status,
		Search:   search,
	})
}

func (s *adminServiceImpl) ListRedeemCodesWithOptions(ctx context.Context, input RedeemCodeListInput) ([]RedeemCode, int64, error) {
	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	if repo, ok := s.redeemCodeRepo.(redeemCodeSortedLister); ok {
		codes, result, err := repo.ListWithFiltersAndSort(ctx, params, input.Type, input.Status, input.Search, input.SortBy, input.SortOrder)
		if err != nil {
			return nil, 0, err
		}
		return codes, result.Total, nil
	}
	codes, result, err := s.redeemCodeRepo.ListWithFilters(ctx, params, input.Type, input.Status, input.Search)
	if err != nil {
		return nil, 0, err
	}
	return codes, result.Total, nil
}
func (s *adminServiceImpl) GetRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	return s.redeemCodeRepo.GetByID(ctx, id)
}
func (s *adminServiceImpl) GenerateRedeemCodes(ctx context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error) {
	if input.Type != RedeemTypeInvitation && input.Value == 0 {
		return nil, errors.New("value must not be zero")
	}
	if input.Type == RedeemTypeSubscription {
		if input.GroupID == nil {
			return nil, errors.New("group_id is required for subscription type")
		}
		if input.ValidityDays == 0 {
			return nil, errors.New("validity_days must not be zero for subscription type")
		}
		group, err := s.groupRepo.GetByID(ctx, *input.GroupID)
		if err != nil {
			return nil, fmt.Errorf("group not found: %w", err)
		}
		if !group.IsSubscriptionType() {
			return nil, errors.New("group must be subscription type")
		}
	}
	codes := make([]RedeemCode, 0, input.Count)
	for i := 0; i < input.Count; i++ {
		codeValue, err := GenerateRedeemCode()
		if err != nil {
			return nil, err
		}
		code := RedeemCode{Code: codeValue, Type: input.Type, Value: input.Value, Status: StatusUnused, ExpiresAt: input.ExpiresAt}
		if input.Type == RedeemTypeSubscription {
			code.GroupID = input.GroupID
			code.ValidityDays = input.ValidityDays
		}
		if err := s.redeemCodeRepo.Create(ctx, &code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}
func (s *adminServiceImpl) DeleteRedeemCode(ctx context.Context, id int64) error {
	return s.redeemCodeRepo.Delete(ctx, id)
}
func (s *adminServiceImpl) BatchDeleteRedeemCodes(ctx context.Context, ids []int64) (int64, error) {
	var deleted int64
	for _, id := range ids {
		if err := s.redeemCodeRepo.Delete(ctx, id); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (s *adminServiceImpl) BatchUpdateRedeemCodes(ctx context.Context, input *BatchUpdateRedeemCodesInput) (int64, error) {
	if input == nil || len(input.IDs) == 0 {
		return 0, infraerrors.BadRequest("REDEEM_CODE_BATCH_IDS_REQUIRED", "ids are required")
	}
	if !input.hasUpdates() {
		return 0, infraerrors.BadRequest("REDEEM_CODE_BATCH_FIELDS_REQUIRED", "fields are required")
	}

	updates := make([]*RedeemCode, 0, len(input.IDs))
	for _, id := range input.IDs {
		code, err := s.redeemCodeRepo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		next := *code
		applyRedeemCodeBatchFields(&next, input)
		if err := s.validateRedeemCodeBatchUpdate(ctx, &next); err != nil {
			return 0, err
		}
		updates = append(updates, &next)
	}

	var updated int64
	for _, code := range updates {
		if err := s.redeemCodeRepo.Update(ctx, code); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

func (s *adminServiceImpl) ExpireRedeemCode(ctx context.Context, id int64) (*RedeemCode, error) {
	code, err := s.redeemCodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	code.Status = StatusExpired
	if err := s.redeemCodeRepo.Update(ctx, code); err != nil {
		return nil, err
	}
	return code, nil
}

func (input *BatchUpdateRedeemCodesInput) hasUpdates() bool {
	if input == nil {
		return false
	}
	return input.Status != nil || input.Notes != nil || input.ExpiresAtSet ||
		input.GroupIDSet || input.Type != nil || input.Value != nil || input.ValidityDays != nil
}

func applyRedeemCodeBatchFields(code *RedeemCode, input *BatchUpdateRedeemCodesInput) {
	if input.Status != nil {
		code.Status = *input.Status
		if code.Status == StatusUnused {
			code.UsedBy = nil
			code.UsedAt = nil
		}
	}
	if input.Notes != nil {
		code.Notes = *input.Notes
	}
	if input.ExpiresAtSet {
		code.ExpiresAt = input.ExpiresAt
	}
	if input.GroupIDSet {
		code.GroupID = input.GroupID
	}
	if input.Type != nil {
		code.Type = *input.Type
	}
	if input.Value != nil {
		code.Value = *input.Value
	}
	if input.ValidityDays != nil {
		code.ValidityDays = *input.ValidityDays
	}
	if code.Type != RedeemTypeSubscription {
		code.GroupID = nil
		code.ValidityDays = 0
	}
	if code.Type == RedeemTypeInvitation {
		code.Value = 0
	}
}

func (s *adminServiceImpl) validateRedeemCodeBatchUpdate(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return infraerrors.BadRequest("REDEEM_CODE_INVALID", "redeem code is required")
	}
	switch code.Status {
	case StatusUnused, StatusExpired, StatusDisabled:
	default:
		return infraerrors.BadRequest("REDEEM_CODE_STATUS_INVALID", "status must be unused, expired, or disabled")
	}
	switch code.Type {
	case RedeemTypeBalance, RedeemTypeConcurrency, RedeemTypeSubscription, RedeemTypeInvitation:
	default:
		return infraerrors.BadRequest("REDEEM_CODE_TYPE_INVALID", "invalid redeem code type")
	}
	if code.Type != RedeemTypeInvitation && code.Value == 0 {
		return infraerrors.BadRequest("REDEEM_CODE_VALUE_INVALID", "value must not be zero")
	}
	if code.Type != RedeemTypeSubscription {
		return nil
	}
	if code.GroupID == nil {
		return infraerrors.BadRequest("REDEEM_CODE_SUBSCRIPTION_FIELDS_REQUIRED", "group_id is required for subscription type")
	}
	if code.ValidityDays == 0 {
		return infraerrors.BadRequest("REDEEM_CODE_SUBSCRIPTION_FIELDS_REQUIRED", "validity_days must not be zero for subscription type")
	}
	if s.groupRepo == nil {
		return nil
	}
	group, err := s.groupRepo.GetByID(ctx, *code.GroupID)
	if err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return infraerrors.BadRequest("REDEEM_CODE_GROUP_INVALID", "group must be subscription type")
	}
	return nil
}
