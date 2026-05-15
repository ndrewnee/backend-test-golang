package money

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var maxAmount = decimal.RequireFromString("9999999999999999.99")

func ParseAmount(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Decimal{}, fmt.Errorf("amount is required")
	}

	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("amount must be a decimal string")
	}
	if !amount.GreaterThan(decimal.Zero) {
		return decimal.Decimal{}, fmt.Errorf("amount must be greater than zero")
	}
	if amount.Exponent() < -2 {
		return decimal.Decimal{}, fmt.Errorf("amount must have at most 2 decimal places")
	}
	if amount.GreaterThan(maxAmount) {
		return decimal.Decimal{}, fmt.Errorf("amount is too large")
	}

	return amount, nil
}

func ParseDatabaseValue(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Decimal{}, fmt.Errorf("database money value is empty")
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parse database money value: %w", err)
	}
	return amount, nil
}

func Format(amount decimal.Decimal) string {
	return amount.StringFixed(2)
}
