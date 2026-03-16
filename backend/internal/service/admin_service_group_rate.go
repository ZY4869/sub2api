package service

import "context"

func (s *adminServiceImpl) GetGroupRateMultipliers(ctx context.Context, groupID int64) ([]UserGroupRateEntry, error) {
	if s == nil || s.userGroupRateRepo == nil {
		return nil, nil
	}
	entries, err := s.userGroupRateRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return entries, nil
}

func (s *adminServiceImpl) ClearGroupRateMultipliers(ctx context.Context, groupID int64) error {
	if s == nil || s.userGroupRateRepo == nil {
		return nil
	}
	return s.userGroupRateRepo.DeleteByGroupID(ctx, groupID)
}

func (s *adminServiceImpl) BatchSetGroupRateMultipliers(ctx context.Context, groupID int64, entries []GroupRateMultiplierInput) error {
	if s == nil || s.userGroupRateRepo == nil {
		return nil
	}
	return s.userGroupRateRepo.SyncGroupRateMultipliers(ctx, groupID, entries)
}
