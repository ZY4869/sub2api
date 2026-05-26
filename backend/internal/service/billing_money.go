package service

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const (
	BillingAmountScale int64 = 100000000
	maxBillingUnits          = int64(999999999999999999)
)

var (
	bigBillingScale = big.NewInt(BillingAmountScale)
	bigMaxBilling   = big.NewInt(maxBillingUnits)
)

// BillingMoney is the fixed-point value object for core billing math.
// Core balance, wallet, hold, ledger, and rebate code should do arithmetic with
// BillingMoney, converting to float64 only at public DTO or legacy ent edges.
type BillingMoney struct {
	units int64
}

func NewBillingMoneyFromFloat(value float64) (BillingMoney, error) {
	if !validBillingAmount(value) || IsNegativeZeroBillingAmount(value) {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	return ParseBillingMoney(strconv.FormatFloat(value, 'f', -1, 64))
}

func NewPositiveBillingMoneyFromFloat(value float64) (BillingMoney, error) {
	money, err := NewBillingMoneyFromFloat(value)
	if err != nil {
		return BillingMoney{}, err
	}
	if !money.IsPositive() {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	return money, nil
}

func NewNonNegativeBillingMoneyFromFloat(value float64) (BillingMoney, error) {
	money, err := NewBillingMoneyFromFloat(value)
	if err != nil {
		return BillingMoney{}, err
	}
	if money.IsNegative() {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	return money, nil
}

func ParseBillingMoney(raw string) (BillingMoney, error) {
	units, err := parseBillingUnits(raw)
	if err != nil {
		return BillingMoney{}, err
	}
	return BillingMoney{units: units}, nil
}

func BillingMoneyFromUnits(units int64) (BillingMoney, error) {
	if err := validateBillingUnits(units); err != nil {
		return BillingMoney{}, err
	}
	return BillingMoney{units: units}, nil
}

func (m BillingMoney) Units() int64 {
	return m.units
}

func (m BillingMoney) Float64() float64 {
	return float64(m.units) / float64(BillingAmountScale)
}

func (m BillingMoney) DBValue() string {
	return m.String()
}

func (m BillingMoney) String() string {
	units := m.units
	sign := ""
	if units < 0 {
		sign = "-"
		units = -units
	}
	whole := units / BillingAmountScale
	frac := units % BillingAmountScale
	if frac == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	fracText := fmt.Sprintf("%08d", frac)
	fracText = strings.TrimRight(fracText, "0")
	return fmt.Sprintf("%s%d.%s", sign, whole, fracText)
}

func (m BillingMoney) IsZero() bool {
	return m.units == 0
}

func (m BillingMoney) IsPositive() bool {
	return m.units > 0
}

func (m BillingMoney) IsNegative() bool {
	return m.units < 0
}

func (m BillingMoney) Cmp(other BillingMoney) int {
	switch {
	case m.units < other.units:
		return -1
	case m.units > other.units:
		return 1
	default:
		return 0
	}
}

func (m BillingMoney) Neg() (BillingMoney, error) {
	return BillingMoneyFromUnits(-m.units)
}

func (m BillingMoney) Add(other BillingMoney) (BillingMoney, error) {
	if (other.units > 0 && m.units > maxBillingUnits-other.units) ||
		(other.units < 0 && m.units < -maxBillingUnits-other.units) {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	return BillingMoneyFromUnits(m.units + other.units)
}

func (m BillingMoney) Sub(other BillingMoney) (BillingMoney, error) {
	neg, err := other.Neg()
	if err != nil {
		return BillingMoney{}, err
	}
	return m.Add(neg)
}

func (m BillingMoney) MulRate(rate float64) (BillingMoney, error) {
	rateMoney, err := NewBillingMoneyFromFloat(rate)
	if err != nil {
		return BillingMoney{}, err
	}
	return roundSignedBigRatio(mulInt64(m.units, rateMoney.units), bigBillingScale)
}

func (m BillingMoney) DivRate(rate float64) (BillingMoney, error) {
	rateMoney, err := NewPositiveBillingMoneyFromFloat(rate)
	if err != nil {
		return BillingMoney{}, err
	}
	return roundSignedBigRatio(mulInt64(m.units, BillingAmountScale), big.NewInt(rateMoney.units))
}

func (m BillingMoney) MulRatePercent(percent float64) (BillingMoney, error) {
	percentMoney, err := NewBillingMoneyFromFloat(percent)
	if err != nil {
		return BillingMoney{}, err
	}
	den := new(big.Int).Mul(big.NewInt(100), bigBillingScale)
	return roundSignedBigRatio(mulInt64(m.units, percentMoney.units), den)
}

func ValidBillingMoneyFloat(value float64) bool {
	if !validBillingAmount(value) || IsNegativeZeroBillingAmount(value) {
		return false
	}
	_, err := NewBillingMoneyFromFloat(value)
	return err == nil
}
