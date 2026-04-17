package service

import "context"

func (s *SettingService) IsDocumentAIEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyDocumentAIEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
