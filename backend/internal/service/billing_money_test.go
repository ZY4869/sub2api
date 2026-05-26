//go:build unit

package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBillingAmount(t *testing.T) {
	require.Equal(t, 0.3, NormalizeBillingAmount(0.1+0.2))
	require.Equal(t, 0.00000001, NormalizeBillingAmount(0.000000014))
	require.Equal(t, 0.0, NormalizeBillingAmount(math.NaN()))
	require.Equal(t, 0.0, NormalizeBillingAmount(math.Inf(1)))
	require.Equal(t, 0.0, NormalizeBillingAmount(-0.0))
}

func TestNormalizeAndValidateBillingAmount(t *testing.T) {
	value, err := NormalizeAndValidateBillingAmount(0.1 + 0.2)
	require.NoError(t, err)
	require.Equal(t, 0.3, value)

	value, err = NormalizeAndValidateBillingAmount(0.000000014)
	require.NoError(t, err)
	require.Equal(t, 0.00000001, value)

	value, err = NormalizeAndValidateBillingAmount(1.234567895)
	require.NoError(t, err)
	require.Equal(t, 1.2345679, value)
}

func TestValidateBillingAmountRejectsUnsafeValues(t *testing.T) {
	for _, value := range []float64{
		math.NaN(),
		math.Inf(1),
		math.Copysign(0, -1),
		MaxBillingAmount + 1,
		-(MaxBillingAmount + 1),
	} {
		_, err := NormalizeAndValidateBillingAmount(value)
		require.ErrorIs(t, err, ErrInvalidBillingAmount)
	}
}

func TestNormalizeAndValidateNonNegativeBillingAmount(t *testing.T) {
	_, err := NormalizeAndValidateNonNegativeBillingAmount(-0.01)
	require.ErrorIs(t, err, ErrInvalidBillingAmount)

	value, err := NormalizeAndValidateNonNegativeBillingAmount(0)
	require.NoError(t, err)
	require.Equal(t, 0.0, value)
}

func TestNormalizeAndValidateBillingAmount_RebateCapBoundary(t *testing.T) {
	accrued, err := NormalizeAndValidateBillingAmount(0.1 + 0.2)
	require.NoError(t, err)
	require.Equal(t, 0.3, accrued)

	remaining, err := NormalizeAndValidateBillingAmount(1.0 - 0.7000000000000001)
	require.NoError(t, err)
	require.Equal(t, 0.3, remaining)

	if accrued > remaining {
		accrued = remaining
	}
	require.Equal(t, 0.3, accrued)
}

func TestBillingMoneyParseAndString(t *testing.T) {
	money, err := ParseBillingMoney("12.340000009")
	require.NoError(t, err)
	require.Equal(t, int64(1234000001), money.Units())
	require.Equal(t, "12.34000001", money.DBValue())

	money, err = ParseBillingMoney(".000000014")
	require.NoError(t, err)
	require.Equal(t, "0.00000001", money.String())

	_, err = ParseBillingMoney("-0.000000004")
	require.ErrorIs(t, err, ErrInvalidBillingAmount)
}

func TestBillingMoneyArithmetic(t *testing.T) {
	a, err := NewBillingMoneyFromFloat(0.1 + 0.2)
	require.NoError(t, err)
	b, err := ParseBillingMoney("0.7")
	require.NoError(t, err)

	sum, err := a.Add(b)
	require.NoError(t, err)
	require.Equal(t, "1", sum.String())

	diff, err := b.Sub(a)
	require.NoError(t, err)
	require.Equal(t, "0.4", diff.String())
	require.Equal(t, 1, b.Cmp(a))
}

func TestBillingMoneyRates(t *testing.T) {
	base, err := ParseBillingMoney("10")
	require.NoError(t, err)

	rebate, err := base.MulRatePercent(12.3456789)
	require.NoError(t, err)
	require.Equal(t, "1.23456789", rebate.String())

	converted, err := rebate.DivRate(7.1)
	require.NoError(t, err)
	require.Equal(t, "0.1738828", converted.String())
}

func TestBillingMoneyRejectsUnsafeValues(t *testing.T) {
	_, err := NewBillingMoneyFromFloat(math.NaN())
	require.ErrorIs(t, err, ErrInvalidBillingAmount)

	_, err = NewBillingMoneyFromFloat(math.Inf(1))
	require.ErrorIs(t, err, ErrInvalidBillingAmount)

	_, err = NewBillingMoneyFromFloat(math.Copysign(0, -1))
	require.ErrorIs(t, err, ErrInvalidBillingAmount)

	_, err = ParseBillingMoney("10000000000")
	require.ErrorIs(t, err, ErrInvalidBillingAmount)
}
