package dto

// ProcessInvoiceResponse is the JSON body returned after a successful
// parse+classify+calculate run. Monetary fields are formatted as decimal
// strings, not JSON numbers, to avoid float round-tripping ambiguity in
// clients.
type ProcessInvoiceResponse struct {
	CUFE          string           `json:"cufe"`
	InvoiceNumber string           `json:"invoice_number"`
	IssuerNIT     string           `json:"issuer_nit"`
	IssuerName    string           `json:"issuer_name"`
	InvoiceTotal  string           `json:"invoice_total"`
	Calculations  []CalculationDTO `json:"calculations"`
}

type CalculationDTO struct {
	InvoiceLineID   uint   `json:"invoice_line_id"`
	TaxType         string `json:"tax_type"`
	ConceptID       *uint  `json:"concept_id"`
	BaseAmount      string `json:"base_amount"`
	TariffApplied   string `json:"tariff_applied"`
	CalculatedValue string `json:"calculated_value"`
	LegalBasis      string `json:"legal_basis"`
	Justification   string `json:"justification"`
}
