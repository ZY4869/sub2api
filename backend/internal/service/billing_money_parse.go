package service

import (
	"math/big"
	"strconv"
	"strings"
)

func parseBillingUnits(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrInvalidBillingAmount
	}
	sign := int64(1)
	if s[0] == '+' || s[0] == '-' {
		if s[0] == '-' {
			sign = -1
		}
		s = strings.TrimSpace(s[1:])
	}
	if s == "" || strings.ContainsAny(s, "eE") {
		return 0, ErrInvalidBillingAmount
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, ErrInvalidBillingAmount
	}
	wholeText := parts[0]
	if wholeText == "" {
		wholeText = "0"
	}
	if !decimalDigitsOnly(wholeText) {
		return 0, ErrInvalidBillingAmount
	}
	fracText := ""
	if len(parts) == 2 {
		fracText = parts[1]
		if fracText != "" && !decimalDigitsOnly(fracText) {
			return 0, ErrInvalidBillingAmount
		}
	}

	whole := new(big.Int)
	if _, ok := whole.SetString(wholeText, 10); !ok {
		return 0, ErrInvalidBillingAmount
	}
	units := new(big.Int).Mul(whole, bigBillingScale)
	fracUnits, roundUp, err := parseFractionalBillingUnits(fracText)
	if err != nil {
		return 0, err
	}
	units.Add(units, big.NewInt(fracUnits))
	if roundUp {
		units.Add(units, big.NewInt(1))
	}
	if sign < 0 {
		units.Neg(units)
	}
	if sign < 0 && units.Sign() == 0 {
		return 0, ErrInvalidBillingAmount
	}
	if new(big.Int).Abs(units).Cmp(bigMaxBilling) > 0 {
		return 0, ErrInvalidBillingAmount
	}
	return units.Int64(), nil
}

func parseFractionalBillingUnits(frac string) (int64, bool, error) {
	if len(frac) > 9 {
		frac = frac[:9]
	}
	roundUp := false
	if len(frac) > 8 {
		roundUp = frac[8] >= '5'
		frac = frac[:8]
	}
	for len(frac) < 8 {
		frac += "0"
	}
	if frac == "" {
		return 0, roundUp, nil
	}
	value, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, false, ErrInvalidBillingAmount
	}
	return value, roundUp, nil
}

func decimalDigitsOnly(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateBillingUnits(units int64) error {
	if units > maxBillingUnits || units < -maxBillingUnits {
		return ErrInvalidBillingAmount
	}
	return nil
}

func mulInt64(left, right int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(left), big.NewInt(right))
}

func roundSignedBigRatio(num *big.Int, den *big.Int) (BillingMoney, error) {
	if den == nil || den.Sign() <= 0 {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	if num == nil || num.Sign() == 0 {
		return BillingMoney{}, nil
	}
	sign := num.Sign()
	absNum := new(big.Int).Abs(new(big.Int).Set(num))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(absNum, den, rem)
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(den) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if q.Cmp(bigMaxBilling) > 0 {
		return BillingMoney{}, ErrInvalidBillingAmount
	}
	units := q.Int64()
	if sign < 0 {
		units = -units
	}
	return BillingMoneyFromUnits(units)
}
