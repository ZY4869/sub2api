package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAPIKeyGroupAdvancedFieldsForbidden = infraerrors.Forbidden(
	"GROUP_BINDING_ADVANCED_FIELDS_FORBIDDEN",
	"only administrators can set group quota or model patterns",
)

type APIKeyGroupUpdateInput struct {
	GroupID       int64
	Quota         float64
	ModelPatterns []string
}

type apiKeyGroupMutationTxStarter interface {
	BeginTx(ctx context.Context) (*dbent.Tx, error)
}

type apiKeyGroupBindingMutationDeps struct {
	groupRepo   GroupRepository
	userRepo    UserRepository
	userSubRepo UserSubscriptionRepository
}

func resolveRequestedAPIKeyGroupUpdates(groupID *int64, groups *[]APIKeyGroupUpdateInput) (*[]APIKeyGroupUpdateInput, error) {
	if groups != nil {
		cloned := make([]APIKeyGroupUpdateInput, 0, len(*groups))
		for _, item := range *groups {
			cloned = append(cloned, APIKeyGroupUpdateInput{
				GroupID:       item.GroupID,
				Quota:         item.Quota,
				ModelPatterns: append([]string(nil), item.ModelPatterns...),
			})
		}
		return &cloned, nil
	}
	if groupID == nil {
		return nil, nil
	}
	if *groupID < 0 {
		return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
	}
	if *groupID == 0 {
		empty := []APIKeyGroupUpdateInput{}
		return &empty, nil
	}
	items := []APIKeyGroupUpdateInput{{GroupID: *groupID}}
	return &items, nil
}

func normalizeAPIKeyGroupUpdateInputs(inputs []APIKeyGroupUpdateInput, allowAdvancedFields bool) ([]APIKeyGroupUpdateInput, error) {
	if len(inputs) == 0 {
		return []APIKeyGroupUpdateInput{}, nil
	}

	seen := make(map[int64]struct{}, len(inputs))
	normalized := make([]APIKeyGroupUpdateInput, 0, len(inputs))
	for _, input := range inputs {
		if input.GroupID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_BINDING", "group_id must be positive")
		}
		if input.Quota < 0 {
			return nil, infraerrors.BadRequest("INVALID_GROUP_BINDING", "quota must be non-negative")
		}
		if _, exists := seen[input.GroupID]; exists {
			return nil, infraerrors.BadRequest("INVALID_GROUP_BINDING", "duplicate group binding")
		}
		seen[input.GroupID] = struct{}{}

		modelPatterns := make([]string, 0, len(input.ModelPatterns))
		patternSeen := make(map[string]struct{}, len(input.ModelPatterns))
		for _, pattern := range input.ModelPatterns {
			trimmed := strings.TrimSpace(pattern)
			if trimmed == "" {
				continue
			}
			if _, exists := patternSeen[trimmed]; exists {
				continue
			}
			patternSeen[trimmed] = struct{}{}
			modelPatterns = append(modelPatterns, trimmed)
		}

		if !allowAdvancedFields && (input.Quota > 0 || len(modelPatterns) > 0) {
			return nil, ErrAPIKeyGroupAdvancedFieldsForbidden
		}

		normalized = append(normalized, APIKeyGroupUpdateInput{
			GroupID:       input.GroupID,
			Quota:         input.Quota,
			ModelPatterns: modelPatterns,
		})
	}

	return normalized, nil
}

func buildAPIKeyGroupBindings(
	ctx context.Context,
	deps apiKeyGroupBindingMutationDeps,
	owner *User,
	apiKeyID int64,
	existingBindings []APIKeyGroupBinding,
	inputs []APIKeyGroupUpdateInput,
	actingAsAdmin bool,
) ([]APIKeyGroupBinding, []AdminGrantedGroupAccess, error) {
	if owner == nil {
		return nil, nil, infraerrors.InternalServer("USER_NOT_LOADED", "user context is required")
	}

	normalizedInputs, err := normalizeAPIKeyGroupUpdateInputs(inputs, actingAsAdmin)
	if err != nil {
		return nil, nil, err
	}

	existingByGroupID := make(map[int64]APIKeyGroupBinding, len(existingBindings))
	for _, binding := range existingBindings {
		existingByGroupID[binding.GroupID] = binding
	}

	bindings := make([]APIKeyGroupBinding, 0, len(normalizedInputs))
	grantedGroups := make([]AdminGrantedGroupAccess, 0)
	for _, input := range normalizedInputs {
		group, err := deps.groupRepo.GetByID(ctx, input.GroupID)
		if err != nil {
			return nil, nil, err
		}
		if !group.IsActive() {
			return nil, nil, infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
		}

		if group.IsSubscriptionType() {
			if deps.userSubRepo == nil {
				return nil, nil, infraerrors.InternalServer("SUBSCRIPTION_REPOSITORY_UNAVAILABLE", "subscription repository is not configured")
			}
			if _, err := deps.userSubRepo.GetActiveByUserIDAndGroupID(ctx, owner.ID, group.ID); err != nil {
				if errors.Is(err, ErrSubscriptionNotFound) {
					if actingAsAdmin {
						return nil, nil, infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
					}
					return nil, nil, ErrGroupNotAllowed
				}
				return nil, nil, err
			}
		} else if actingAsAdmin && group.IsExclusive {
			if deps.userRepo == nil {
				return nil, nil, infraerrors.InternalServer("USER_REPOSITORY_UNAVAILABLE", "user repository is not configured")
			}
			if err := deps.userRepo.AddGroupToAllowedGroups(ctx, owner.ID, group.ID); err != nil {
				return nil, nil, fmt.Errorf("add group to user allowed groups: %w", err)
			}
			grantedGroups = append(grantedGroups, AdminGrantedGroupAccess{
				GroupID:   group.ID,
				GroupName: group.Name,
			})
		} else if !actingAsAdmin && !owner.CanBindGroup(group.ID, group.IsExclusive) {
			return nil, nil, ErrGroupNotAllowed
		}

		binding := APIKeyGroupBinding{
			APIKeyID:      apiKeyID,
			GroupID:       group.ID,
			Group:         group,
			Quota:         input.Quota,
			ModelPatterns: append([]string(nil), input.ModelPatterns...),
		}
		if existing, ok := existingByGroupID[group.ID]; ok {
			binding.QuotaUsed = existing.QuotaUsed
		}
		bindings = append(bindings, binding)
	}

	return bindings, grantedGroups, nil
}

func beginAPIKeyMutationTx(ctx context.Context, starter apiKeyGroupMutationTxStarter) (context.Context, *dbent.Tx, func(), error) {
	if starter == nil {
		return ctx, nil, func() {}, nil
	}
	tx, err := starter.BeginTx(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return dbent.NewTxContext(ctx, tx), tx, func() {
		_ = tx.Rollback()
	}, nil
}
