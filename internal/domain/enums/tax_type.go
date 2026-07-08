package enums

// TaxType are the official DIAN withholding tax acronyms, kept untranslated
// since they are the exact legal names used in Colombian tax regulation.
type TaxType string

const (
	TaxTypeRetefuente TaxType = "RETEFUENTE"
	TaxTypeReteiva    TaxType = "RETEIVA"
	TaxTypeReteica    TaxType = "RETEICA"
)
