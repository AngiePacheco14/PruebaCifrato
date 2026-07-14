package dto

// ProcessInvoiceResponse is the JSON body returned after a successful
// parse+classify+calculate run. Monetary fields are decimal strings, not
// JSON numbers, to avoid float round-tripping issues.
type ProcessInvoiceResponse struct {
	CUFE          string           `json:"cufe"`
	InvoiceNumber string           `json:"invoice_number"`
	IssuerNIT     string           `json:"issuer_nit"`
	IssuerName    string           `json:"issuer_name"`
	InvoiceTotal  string           `json:"invoice_total"`
	Summary       SummaryDTO       `json:"summary"`
	Calculations  []CalculationDTO `json:"calculations"`
}

// SummaryDTO is the invoice-level rollup: CalculatedValue summed per tax type.
type SummaryDTO struct {
	TotalRetefuente string `json:"total_retefuente"`
	TotalReteiva    string `json:"total_reteiva"`
	TotalReteica    string `json:"total_reteica"`
}

// CalculationDTO is one tax type's result for one concept. BaseAmount sums
// every line under that concept, not a single line's amount.
type CalculationDTO struct {
	TaxType         string  `json:"tax_type"`
	ConceptID       *uint   `json:"concept_id"`
	ConceptName     *string `json:"concept_name"`
	BaseAmount      string  `json:"base_amount"`
	TariffApplied   string  `json:"tariff_applied"`
	CalculatedValue string  `json:"calculated_value"`
	LegalBasis      string  `json:"legal_basis"`
	Justification   string  `json:"justification"`
}
