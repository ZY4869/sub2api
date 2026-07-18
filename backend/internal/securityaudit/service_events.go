package securityaudit

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *PromptService) ListEvents(ctx context.Context, filter EventFilter) (*EventList, error) {
	return s.repo.ListEvents(ctx, filter)
}

func (s *PromptService) GetEvent(ctx context.Context, id int64) (*Event, error) {
	return s.repo.GetEvent(ctx, id, true)
}

func (s *PromptService) DeleteEvent(ctx context.Context, id int64) (*DeleteResult, error) {
	result, err := s.repo.DeleteEvent(ctx, id)
	if err == nil {
		deletePayloads(ctx, s.payload, result.JobIDs)
	}
	return result, err
}

func (s *PromptService) PreviewDelete(ctx context.Context, filter EventFilter, adminID int64) (*DeletePreview, error) {
	preview, err := s.repo.PreviewDelete(ctx, filter)
	if err != nil {
		return nil, err
	}
	if err := s.attachDeleteConfirmation(preview, adminID); err != nil {
		return nil, err
	}
	return preview, nil
}

func (s *PromptService) DeleteByFilter(ctx context.Context, req DeleteByFilterRequest, adminID int64) (*DeleteResult, error) {
	if !req.Confirm {
		return nil, infraerrors.BadRequest("prompt_audit_delete_confirmation_invalid", "delete confirmation invalid")
	}
	if err := s.verifyDeleteToken(req, adminID); err != nil {
		return nil, err
	}
	result, err := s.repo.DeleteEventsByFilter(ctx, req.Filter, req.SnapshotMaxID)
	if err == nil {
		deletePayloads(ctx, s.payload, result.JobIDs)
	}
	return result, err
}

func (s *PromptService) attachDeleteConfirmation(preview *DeletePreview, adminID int64) error {
	preview.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)
	claims := map[string]any{
		"filter_hash": preview.FilterHash, "snapshot_max_id": preview.SnapshotMaxID,
		"admin_id": adminID, "exp": preview.ExpiresAt.Unix(),
	}
	raw, _ := json.Marshal(claims)
	token, err := s.config.Encrypt(string(raw))
	if err != nil {
		return err
	}
	preview.ConfirmationToken = token
	return nil
}

func (s *PromptService) verifyDeleteToken(req DeleteByFilterRequest, adminID int64) error {
	plain, err := s.config.Decrypt(strings.TrimSpace(req.ConfirmationToken))
	if err != nil {
		return deleteConfirmationError()
	}
	var claims deleteConfirmationClaims
	if json.Unmarshal([]byte(plain), &claims) != nil {
		return deleteConfirmationError()
	}
	if !validDeleteClaims(claims, req, adminID) {
		return deleteConfirmationError()
	}
	return nil
}

func validDeleteClaims(claims deleteConfirmationClaims, req DeleteByFilterRequest, adminID int64) bool {
	expectedHash := filterHash(normalizeEventFilter(req.Filter), req.SnapshotMaxID)
	return claims.AdminID == adminID &&
		claims.SnapshotMaxID == req.SnapshotMaxID &&
		claims.FilterHash == expectedHash &&
		claims.ExpiresAt > time.Now().Unix()
}

func deleteConfirmationError() error {
	return infraerrors.BadRequest("prompt_audit_delete_confirmation_invalid", "delete confirmation invalid")
}

func deletePayloads(ctx context.Context, store PayloadStore, jobIDs []int64) {
	for _, id := range jobIDs {
		_ = store.Delete(ctx, id)
	}
}
