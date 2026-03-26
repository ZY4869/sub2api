package service

import "testing"

func TestDetectHardBannedAccountMatchesWorkspaceDeactivatedTopLevelCode(t *testing.T) {
	t.Parallel()

	match := DetectHardBannedAccount(402, []byte(`{"code":"deactivated_workspace","message":"Workspace is deactivated"}`))
	if match == nil {
		t.Fatal("expected hard ban match, got nil")
	}
	if match.ReasonCode != "workspace_deactivated" {
		t.Fatalf("ReasonCode = %q, want %q", match.ReasonCode, "workspace_deactivated")
	}
	if match.ReasonMessage != "Workspace is deactivated" {
		t.Fatalf("ReasonMessage = %q, want %q", match.ReasonMessage, "Workspace is deactivated")
	}
}

func TestDetectHardBannedAccountMatchesWorkspaceDeactivatedNestedCode(t *testing.T) {
	t.Parallel()

	match := DetectHardBannedAccount(402, []byte(`{"error":{"code":"deactivated_workspace","message":"Workspace access disabled"}}`))
	if match == nil {
		t.Fatal("expected hard ban match, got nil")
	}
	if match.ReasonCode != "workspace_deactivated" {
		t.Fatalf("ReasonCode = %q, want %q", match.ReasonCode, "workspace_deactivated")
	}
	if match.ReasonMessage != "Workspace access disabled" {
		t.Fatalf("ReasonMessage = %q, want %q", match.ReasonMessage, "Workspace access disabled")
	}
}

func TestDetectHardBannedAccountIgnoresGenericPaymentRequired(t *testing.T) {
	t.Parallel()

	match := DetectHardBannedAccount(402, []byte(`{"error":{"code":"insufficient_quota","message":"Billing issue"}}`))
	if match != nil {
		t.Fatalf("expected no hard ban match, got %+v", match)
	}
}

func TestDetectHardBannedAccountPreservesExistingDeactivatedSignals(t *testing.T) {
	t.Parallel()

	match := DetectHardBannedAccount(403, []byte(`{"error":{"code":"account_deactivated","message":"Account disabled"}}`))
	if match == nil {
		t.Fatal("expected hard ban match, got nil")
	}
	if match.ReasonCode != "account_deactivated" {
		t.Fatalf("ReasonCode = %q, want %q", match.ReasonCode, "account_deactivated")
	}
}
