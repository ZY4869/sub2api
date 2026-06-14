package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	adminComplianceDocumentVersion = "2026-06-14"
	adminComplianceDocumentText    = "Administrators are responsible for lawful, authorized, and privacy-preserving operation of this deployment."
)

type AdminComplianceStatus struct {
	Enabled         bool       `json:"enabled"`
	Required        bool       `json:"required"`
	DocumentVersion string     `json:"document_version"`
	DocumentHash    string     `json:"document_hash"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
}

type adminComplianceAcknowledgement struct {
	UserID          int64     `json:"user_id"`
	DocumentVersion string    `json:"document_version"`
	DocumentHash    string    `json:"document_hash"`
	AcknowledgedAt  time.Time `json:"acknowledged_at"`
}

func adminComplianceAcknowledgementKey(userID int64) string {
	return "admin_compliance_acknowledgement:" + strconv.FormatInt(userID, 10)
}

func AdminComplianceDocumentHash() string {
	sum := sha256.Sum256([]byte(adminComplianceDocumentVersion + "\n" + adminComplianceDocumentText))
	return hex.EncodeToString(sum[:])
}

func (s *SettingService) IsAdminComplianceEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAdminComplianceEnabled)
	if err != nil {
		return false
	}
	return parseSettingBool(value, false)
}

func (s *SettingService) GetAdminComplianceStatus(ctx context.Context, userID int64) (*AdminComplianceStatus, error) {
	enabled := s.IsAdminComplianceEnabled(ctx)
	status := &AdminComplianceStatus{
		Enabled:         enabled,
		DocumentVersion: adminComplianceDocumentVersion,
		DocumentHash:    AdminComplianceDocumentHash(),
	}
	if !enabled || userID <= 0 || s == nil || s.settingRepo == nil {
		status.Required = enabled
		return status, nil
	}

	ack, err := s.getAdminComplianceAcknowledgement(ctx, userID)
	if err != nil {
		return nil, err
	}
	if ack != nil && ack.DocumentVersion == status.DocumentVersion && ack.DocumentHash == status.DocumentHash {
		status.AcknowledgedAt = &ack.AcknowledgedAt
	}
	status.Required = status.AcknowledgedAt == nil
	return status, nil
}

func (s *SettingService) AcknowledgeAdminCompliance(ctx context.Context, userID int64) (*AdminComplianceStatus, error) {
	if userID <= 0 {
		return nil, ErrInsufficientPerms
	}
	if s == nil || s.settingRepo == nil {
		return nil, ErrSettingNotFound
	}
	ack := adminComplianceAcknowledgement{
		UserID:          userID,
		DocumentVersion: adminComplianceDocumentVersion,
		DocumentHash:    AdminComplianceDocumentHash(),
		AcknowledgedAt:  time.Now().UTC(),
	}
	raw, err := json.Marshal(ack)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, adminComplianceAcknowledgementKey(userID), string(raw)); err != nil {
		return nil, err
	}
	return s.GetAdminComplianceStatus(ctx, userID)
}

func (s *SettingService) getAdminComplianceAcknowledgement(ctx context.Context, userID int64) (*adminComplianceAcknowledgement, error) {
	value, err := s.settingRepo.GetValue(ctx, adminComplianceAcknowledgementKey(userID))
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil, nil
		}
		return nil, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var ack adminComplianceAcknowledgement
	if err := json.Unmarshal([]byte(value), &ack); err != nil {
		return nil, nil
	}
	if ack.UserID != userID {
		return nil, nil
	}
	return &ack, nil
}
