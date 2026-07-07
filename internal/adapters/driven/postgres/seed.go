package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cifrato/internal/adapters/driven/postgres/models"
)

// Seed inserts minimal reference data to verify the schema and FK wiring
// end-to-end: cities, concepts, and the UVT value. SeedTaxRules (below)
// loads the actual RETEFUENTE/RETEIVA/RETEICA tariff catalog.
func Seed(db *gorm.DB) error {
	cities := []models.CityModel{
		{Name: "BOGOTA D.C", Department: "Bogotá D.C."},
		{Name: "MEDELLIN", Department: "Antioquia"},
		{Name: "GIRARDOTA", Department: "Antioquia"},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&cities).Error; err != nil {
		return err
	}

	concepts := []models.WithholdingConceptModel{
		{Code: "compra_bienes", Name: "Compra de bienes"},
		{Code: "servicios_generales", Name: "Servicios generales"},
		{Code: "transporte_carga", Name: "Transporte de carga"},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&concepts).Error; err != nil {
		return err
	}

	// EffectiveFrom is 2025-01-01, not the 2026 calendar year the UVT value
	// itself belongs to: the sample invoices used for testing span from
	// 2025-08 to 2026-04, and reusing the 2026 UVT value as an approximation
	// for that whole range is a deliberate simplification for this technical
	// test (the real 2025 UVT value differs) — it keeps every sample invoice
	// processable end-to-end instead of failing the ones dated before 2026.
	uvt := models.UVTValueModel{
		Year:                2026,
		Value:               decimal.NewFromInt(52374),
		EffectiveFrom:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		ResolutionReference: "Resolución DIAN 000238 del 15 de diciembre de 2025",
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&uvt).Error; err != nil {
		return err
	}

	return nil
}

// SeedTaxRules loads the RETEFUENTE/RETEIVA/RETEICA tariff catalog used to
// test the withholding engine end-to-end against real sample invoices.
//
// RETEFUENTE compra_bienes/servicios_generales and RETEIVA are confirmed
// figures (contexto-prueba-cifrato.md: Art. 401/392 E.T., Decreto 572 de
// 2025 bases; Art. 437-2 E.T. 15% tariff). RETEFUENTE transporte_carga and
// every RETEICA row are representative EXAMPLE tariffs, not verified
// against an accountant or the current municipal Acuerdo — the source
// document only gives a general legal range (Ley 1819/2016 art. 342, and a
// Bogotá-specific range with no per-activity breakdown), with no figures at
// all for Medellín or Girardota. These MUST be confirmed before any
// production use; see the LegalBasis text on each RETEICA row.
//
// EffectiveFrom is 2025-01-01 for every row (see the same note on Seed's
// UVT insertion) so all 10 sample invoices — the earliest dated 2025-08-03
// — can be processed end-to-end by the engine.
func SeedTaxRules(db *gorm.DB) error {
	concepts, err := conceptIDsByCode(db, "compra_bienes", "servicios_generales", "transporte_carga")
	if err != nil {
		return err
	}
	cities, err := cityIDsByName(db, "BOGOTA D.C", "MEDELLIN", "GIRARDOTA")
	if err != nil {
		return err
	}

	eff := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	icaLegalBasis := "Ley 14 de 1983; Decreto 1333 de 1986; Ley 1819 de 2016 art. 342 — TARIFA DE EJEMPLO PARA PRUEBAS, NO VERIFICADA CONTRA EL ACUERDO MUNICIPAL VIGENTE"

	rows := []models.AdditionalTaxRuleModel{
		// RETEFUENTE nacional — confirmadas en el .md salvo transporte_carga.
		{TaxType: "RETEFUENTE", ConceptID: concepts["compra_bienes"], CityID: nil,
			MinBaseUVT: decimal.NewFromInt(10), TariffPercentage: decimal.NewFromFloat(2.5),
			LegalBasis: "Art. 401 E.T.; Decreto 572 de 2025", EffectiveFrom: eff},
		{TaxType: "RETEFUENTE", ConceptID: concepts["servicios_generales"], CityID: nil,
			MinBaseUVT: decimal.NewFromInt(2), TariffPercentage: decimal.NewFromInt(4),
			LegalBasis: "Art. 392 E.T.; Decreto 572 de 2025", EffectiveFrom: eff},
		{TaxType: "RETEFUENTE", ConceptID: concepts["transporte_carga"], CityID: nil,
			MinBaseUVT: decimal.NewFromInt(4), TariffPercentage: decimal.NewFromInt(1),
			LegalBasis: "Art. 401 E.T. (aplicación análoga) — NO CONFIRMADO EN LA FUENTE, verificar con contador antes de producción", EffectiveFrom: eff},

		// RETEIVA nacional — sin mínimo por decisión de negocio, 15% confirmado.
		{TaxType: "RETEIVA", ConceptID: concepts["compra_bienes"], CityID: nil,
			MinBaseUVT: decimal.Zero, TariffPercentage: decimal.NewFromInt(15),
			LegalBasis: "Art. 437-2 E.T.", EffectiveFrom: eff},
		{TaxType: "RETEIVA", ConceptID: concepts["servicios_generales"], CityID: nil,
			MinBaseUVT: decimal.Zero, TariffPercentage: decimal.NewFromInt(15),
			LegalBasis: "Art. 437-2 E.T.", EffectiveFrom: eff},
		{TaxType: "RETEIVA", ConceptID: concepts["transporte_carga"], CityID: nil,
			MinBaseUVT: decimal.Zero, TariffPercentage: decimal.NewFromInt(15),
			LegalBasis: "Art. 437-2 E.T.", EffectiveFrom: eff},
	}

	// RETEICA — mismas tarifas de ejemplo en las 3 ciudades: no hay dato real
	// que las diferencie (ver comentario de la función). Un loop en vez de
	// 9 filas escritas a mano: la tabla de tarifas por concepto queda en un
	// solo lugar en vez de repetida 3 veces.
	icaTariffsByConcept := []struct {
		concept string
		tariff  float64
	}{
		{"compra_bienes", 0.5},
		{"servicios_generales", 0.7},
		{"transporte_carga", 0.6},
	}
	for _, cityName := range []string{"BOGOTA D.C", "MEDELLIN", "GIRARDOTA"} {
		for _, ct := range icaTariffsByConcept {
			rows = append(rows, models.AdditionalTaxRuleModel{
				TaxType: "RETEICA", ConceptID: concepts[ct.concept], CityID: ptr(cities[cityName]),
				MinBaseUVT: decimal.Zero, TariffPercentage: decimal.NewFromFloat(ct.tariff),
				LegalBasis: icaLegalBasis, EffectiveFrom: eff,
			})
		}
	}

	for i := range rows {
		if err := insertRuleIfMissing(db, &rows[i]); err != nil {
			return err
		}
	}
	return nil
}

func ptr(v uint) *uint { return &v }

func conceptIDsByCode(db *gorm.DB, codes ...string) (map[string]uint, error) {
	var rows []models.WithholdingConceptModel
	if err := db.Where("code IN ?", codes).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: loading concepts for seeding tax rules: %w", err)
	}
	out := make(map[string]uint, len(rows))
	for _, r := range rows {
		out[r.Code] = r.ID
	}
	return out, nil
}

func cityIDsByName(db *gorm.DB, names ...string) (map[string]uint, error) {
	var rows []models.CityModel
	if err := db.Where("name IN ?", names).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: loading cities for seeding tax rules: %w", err)
	}
	out := make(map[string]uint, len(rows))
	for _, r := range rows {
		out[r.Name] = r.ID
	}
	return out, nil
}

// insertRuleIfMissing checks for an existing row before inserting: unlike
// Seed's OnConflict{DoNothing} inserts, additional_taxes_rules has no
// uniqueIndex to conflict on (its lookup index is non-unique, by design —
// see model_withholding.go), so idempotency here is a manual SELECT first.
func insertRuleIfMissing(db *gorm.DB, row *models.AdditionalTaxRuleModel) error {
	q := db.Where("tax_type = ? AND concept_id = ? AND effective_from = ?", row.TaxType, row.ConceptID, row.EffectiveFrom)
	if row.CityID != nil {
		q = q.Where("city_id = ?", *row.CityID)
	} else {
		q = q.Where("city_id IS NULL")
	}

	var existing models.AdditionalTaxRuleModel
	err := q.First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("postgres: checking existing tax rule: %w", err)
	}
	if err := db.Create(row).Error; err != nil {
		return fmt.Errorf("postgres: inserting tax rule: %w", err)
	}
	return nil
}
