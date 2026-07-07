package out

import (
	"context"
	"time"

	"cifrato/internal/domain/withholding"
)

type TaxRuleRepository interface {
	// FindApplicable returns the tax rule valid at the given date for the
	// given concept. cityID nil looks up a national rule (RETEFUENTE/RETEIVA);
	// cityID set looks up a territorial ICA rule for that city.
	FindApplicable(ctx context.Context, taxType withholding.TaxType, conceptID uint, cityID *uint, at time.Time) (*withholding.TaxRule, error)
	ListByTaxType(ctx context.Context, taxType withholding.TaxType) ([]withholding.TaxRule, error)
}
