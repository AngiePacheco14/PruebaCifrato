package postgres

import (
	"gorm.io/gorm"

	"cifrato/internal/adapters/driven/postgres/models"
)

// Migrate runs AutoMigrate in FK-dependency order: reference catalogs first,
// then invoices, then tables that reference invoice lines.
//
// AutoMigrate (not golang-migrate) is a deliberate choice: the Go struct is
// the single source of truth, there is no risk of code/schema drift, and it
// fits a short technical-test timeline with a single environment. Isolated
// here so swapping to versioned SQL migrations later is a contained change.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.CityModel{},
		&models.WithholdingConceptModel{},
		&models.UVTValueModel{},
		&models.InvoiceModel{},
		&models.InvoiceLineModel{},
		&models.AdditionalTaxRuleModel{},
		&models.LineClassificationModel{},
		&models.WithholdingCalculationModel{},
	)
}
