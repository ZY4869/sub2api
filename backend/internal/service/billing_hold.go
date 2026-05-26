package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const DefaultMinimumRequestHoldUSD = 0.01

var ErrBillingHoldNotFound = errors.New("billing request hold not found")

var ErrInvalidBillingAmount = infraerrors.BadRequest("INVALID_BILLING_AMOUNT", "invalid billing amount")
var ErrBillingHoldAlreadyFinished = infraerrors.Conflict("BILLING_REQUEST_REPLAYED", "billing request hold is already finished")
var ErrBillingRequestReplayed = infraerrors.Conflict("BILLING_REQUEST_REPLAYED", "billing request was already used with different payload")

type BillingHoldStatus string

const (
	BillingHoldStatusHeld      BillingHoldStatus = "held"
	BillingHoldStatusSettled   BillingHoldStatus = "settled"
	BillingHoldStatusReleased  BillingHoldStatus = "released"
	BillingHoldStatusCancelled BillingHoldStatus = "cancelled"
)

type BillingHold struct {
	RequestID          string
	APIKeyID           int64
	UserID             int64
	Currency           string
	Amount             float64
	Status             BillingHoldStatus
	RequestFingerprint string
}

type BillingHoldRepository interface {
	Reserve(ctx context.Context, hold *BillingHold) (*BillingHold, error)
	Settle(ctx context.Context, requestID string, apiKeyID int64, actualAmount float64) (*BillingHold, error)
	Release(ctx context.Context, requestID string, apiKeyID int64) (*BillingHold, error)
}

func MinimumRequestHoldUSD(cfg *config.Config) float64 {
	if cfg != nil && cfg.Billing.MinimumRequestHoldUSD > 0 {
		if amount, err := NormalizeAndValidatePositiveBillingAmount(cfg.Billing.MinimumRequestHoldUSD); err == nil {
			return amount
		}
	}
	return DefaultMinimumRequestHoldUSD
}

func BillingHoldSettlementMaxAge(cfg *config.Config) time.Duration {
	if cfg != nil && cfg.Billing.RequestHoldSettlementMaxSeconds > 0 {
		return time.Duration(cfg.Billing.RequestHoldSettlementMaxSeconds) * time.Second
	}
	return 30 * time.Second
}

func NormalizeBillingHoldRequestID(value string) string {
	return strings.TrimSpace(value)
}

func NormalizeBillingHoldRequestFingerprint(value string) string {
	return strings.TrimSpace(value)
}

func BillingHoldRequestFingerprintFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if payloadHash, _ := ctx.Value(ctxkey.RequestPayloadHash).(string); strings.TrimSpace(payloadHash) != "" {
		return strings.TrimSpace(payloadHash)
	}
	return ""
}

func DeductBillingHoldFromUserSnapshot(user *User, hold *BillingHold) {
	if user == nil || hold == nil || hold.Status != BillingHoldStatusHeld || hold.Amount <= 0 {
		return
	}
	holdMoney, err := NewPositiveBillingMoneyFromFloat(hold.Amount)
	if err != nil {
		return
	}
	if balanceMoney, err := NewBillingMoneyFromFloat(user.Balance); err == nil && balanceMoney.IsPositive() {
		next, err := balanceMoney.Sub(holdMoney)
		if err == nil {
			user.Balance = next.Float64()
		}
	}
	if user.Balances == nil {
		return
	}
	if usdMoney, err := NewBillingMoneyFromFloat(user.Balances[ModelPricingCurrencyUSD]); err == nil && usdMoney.IsPositive() {
		next, err := usdMoney.Sub(holdMoney)
		if err == nil {
			user.Balances[ModelPricingCurrencyUSD] = next.Float64()
		}
	}
}
