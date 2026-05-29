package service

import (
	"context"
	"fmt"
)

func (s *APIKeyService) prepareAPIKeyUpdateGroupBindings(
	ctx context.Context,
	apiKey *APIKey,
	user *User,
	requestedGroups *[]APIKeyGroupUpdateInput,
) ([]APIKeyGroupBinding, bool, error) {
	if requestedGroups != nil {
		return s.prepareRequestedAPIKeyUpdateGroupBindings(ctx, apiKey, user, *requestedGroups)
	}
	if !apiKey.ImageOnlyEnabled {
		return nil, false, nil
	}
	bindings, err := s.normalizeImageOnlyGroupBindings(ctx, apiKeyBindingsForSelection(apiKey))
	if err != nil {
		return nil, false, err
	}
	if !user.IsAdmin() {
		if err := s.validateUserAPIKeyModelBindings(ctx, user, bindings); err != nil {
			return nil, false, err
		}
	}
	return bindings, true, nil
}

func (s *APIKeyService) prepareRequestedAPIKeyUpdateGroupBindings(
	ctx context.Context,
	apiKey *APIKey,
	user *User,
	requestedGroups []APIKeyGroupUpdateInput,
) ([]APIKeyGroupBinding, bool, error) {
	existingBindings, err := s.apiKeyRepo.GetAPIKeyGroups(ctx, apiKey.ID)
	if err != nil {
		return nil, false, fmt.Errorf("get api key groups: %w", err)
	}
	bindings, _, err := buildAPIKeyGroupBindings(
		ctx,
		apiKeyGroupBindingMutationDeps{
			groupRepo:   s.groupRepo,
			userRepo:    s.userRepo,
			userSubRepo: s.userSubRepo,
		},
		user,
		apiKey.ID,
		existingBindings,
		requestedGroups,
		user.IsAdmin(),
	)
	if err != nil {
		return nil, false, err
	}
	bindings, err = s.validateAndNormalizeAPIKeyUpdateBindings(ctx, apiKey, user, bindings)
	if err != nil {
		return nil, false, err
	}
	return bindings, true, nil
}

func (s *APIKeyService) validateAndNormalizeAPIKeyUpdateBindings(ctx context.Context, apiKey *APIKey, user *User, bindings []APIKeyGroupBinding) ([]APIKeyGroupBinding, error) {
	if !user.IsAdmin() {
		if err := s.validateUserAPIKeyModelBindings(ctx, user, bindings); err != nil {
			return nil, err
		}
	}
	if !apiKey.ImageOnlyEnabled {
		return bindings, nil
	}
	bindings, err := s.normalizeImageOnlyGroupBindings(ctx, bindings)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin() {
		if err := s.validateUserAPIKeyModelBindings(ctx, user, bindings); err != nil {
			return nil, err
		}
	}
	return bindings, nil
}
