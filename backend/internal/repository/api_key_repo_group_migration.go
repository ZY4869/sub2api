package repository

import "context"

func (r *apiKeyRepository) UpdateGroupIDByUserAndGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (int64, error) {
	exec, commit, rollback, err := beginAPIKeyGroupSQLTx(ctx, apiKeyGroupSQLExecutor(ctx, r))
	if err != nil {
		return 0, err
	}
	defer rollback()

	keyIDs, err := listAPIKeyIDsByUserAndGroup(ctx, exec, userID, oldGroupID)
	if err != nil {
		return 0, err
	}
	if len(keyIDs) == 0 {
		return 0, nil
	}
	if _, err := exec.ExecContext(ctx, `
		DELETE FROM api_key_groups ag
		USING api_keys ak
		WHERE ag.api_key_id = ak.id
		  AND ak.user_id = $1
		  AND ak.deleted_at IS NULL
		  AND ag.group_id = $2
	`, userID, oldGroupID); err != nil {
		return 0, err
	}
	for _, keyID := range keyIDs {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO api_key_groups (api_key_id, group_id, quota, quota_used, model_patterns, created_at, updated_at)
			VALUES ($1, $2, 0, 0, '[]'::jsonb, NOW(), NOW())
			ON CONFLICT (api_key_id, group_id) DO NOTHING
		`, keyID, newGroupID); err != nil {
			return 0, err
		}
	}
	for _, keyID := range keyIDs {
		if err := recomputeAPIKeyShadowGroup(ctx, exec, keyID); err != nil {
			return 0, err
		}
	}
	if err := commit(); err != nil {
		return 0, err
	}
	return int64(len(keyIDs)), nil
}
