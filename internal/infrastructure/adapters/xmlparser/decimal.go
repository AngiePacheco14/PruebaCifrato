package xmlparser

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// parseDecimalRequired parses raw as a decimal.Decimal, returning an error
// with fieldName context when raw is empty or not a valid number.
func parseDecimalRequired(raw, fieldName string) (decimal.Decimal, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return decimal.Decimal{}, fmt.Errorf("missing required field %s", fieldName)
	}
	d, err := decimal.NewFromString(trimmed)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parsing %s value %q: %w", fieldName, raw, err)
	}
	return d, nil
}

// parseDecimalOptional parses raw as a decimal.Decimal, degrading to zero
// (never erroring) when raw is empty or malformed.
func parseDecimalOptional(raw string) decimal.Decimal {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(trimmed)
	if err != nil {
		return decimal.Zero
	}
	return d
}
