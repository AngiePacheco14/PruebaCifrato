package xmlparser

import "encoding/xml"

// Intermediate XML structs. Tags use local element names only (no
// namespace), which lets encoding/xml match regardless of which prefix
// (cbc:/cac:) a given document declares. All monetary/quantity fields are
// kept as string and converted explicitly via decimal.go — encoding/xml has
// no native support for decimal.Decimal.

// --- Root: direct Invoice ---

type xmlInvoice struct {
	XMLName                 xml.Name                 `xml:"Invoice"`
	UUID                    string                   `xml:"UUID"`
	ID                      string                   `xml:"ID"`
	IssueDate               string                   `xml:"IssueDate"`
	AccountingSupplierParty xmlAccountingParty       `xml:"AccountingSupplierParty"`
	AccountingCustomerParty xmlAccountingParty       `xml:"AccountingCustomerParty"`
	TaxTotal                []xmlTaxTotal            `xml:"TaxTotal"`
	WithholdingTaxTotal     []xmlWithholdingTaxTotal `xml:"WithholdingTaxTotal"`
	LegalMonetaryTotal      xmlLegalMonetaryTotal    `xml:"LegalMonetaryTotal"`
	InvoiceLine             []xmlInvoiceLine         `xml:"InvoiceLine"`
}

// --- Root: AttachedDocument. Only Attachment (direct child of the root) is
// mapped; ParentDocumentLineReference is intentionally NOT defined here, so
// encoding/xml silently ignores the nested ApplicationResponse CDATA that
// lives under it — no content sniffing required. ---

type xmlAttachedDocument struct {
	XMLName    xml.Name           `xml:"AttachedDocument"`
	Attachment xmlAttachmentBlock `xml:"Attachment"`
}

type xmlAttachmentBlock struct {
	ExternalReference xmlExternalReference `xml:"ExternalReference"`
}

type xmlExternalReference struct {
	Description string `xml:"Description"` // CDATA text = the full embedded <Invoice> document
}

// --- Party (reused for AccountingSupplierParty / AccountingCustomerParty) ---

type xmlAccountingParty struct {
	Party xmlPartyDetail `xml:"Party"`
}

type xmlPartyDetail struct {
	PhysicalLocation xmlPhysicalLocation `xml:"PhysicalLocation"`
	PartyTaxScheme   xmlPartyTaxScheme   `xml:"PartyTaxScheme"`
	// PartyName is intentionally not mapped: only PartyTaxScheme/RegistrationName
	// (the legal name) is used, since it can differ from the commercial name.
}

type xmlPhysicalLocation struct {
	Address xmlAddress `xml:"Address"`
}

type xmlAddress struct {
	CityName string `xml:"CityName"`
}

type xmlPartyTaxScheme struct {
	RegistrationName    string                 `xml:"RegistrationName"`
	CompanyID           string                 `xml:"CompanyID"`
	TaxLevelCode        string                 `xml:"TaxLevelCode"`
	RegistrationAddress xmlRegistrationAddress `xml:"RegistrationAddress"`
}

type xmlRegistrationAddress struct {
	CityName string `xml:"CityName"`
}

// --- Tax (reused identically for header TaxTotal and line-level TaxTotal) ---

type xmlTaxTotal struct {
	TaxSubtotal []xmlTaxSubtotal `xml:"TaxSubtotal"`
}

type xmlTaxSubtotal struct {
	TaxAmount   string         `xml:"TaxAmount"`
	TaxCategory xmlTaxCategory `xml:"TaxCategory"`
}

type xmlTaxCategory struct {
	Percent   string       `xml:"Percent"`
	TaxScheme xmlTaxScheme `xml:"TaxScheme"`
}

type xmlTaxScheme struct {
	ID   string `xml:"ID"`
	Name string `xml:"Name"`
}

// --- WithholdingTaxTotal (header level only) ---

type xmlWithholdingTaxTotal struct {
	TaxAmount   string           `xml:"TaxAmount"` // amount at the block level, per spec
	TaxSubtotal []xmlTaxSubtotal `xml:"TaxSubtotal"`
}

// --- Monetary totals ---

type xmlLegalMonetaryTotal struct {
	LineExtensionAmount string `xml:"LineExtensionAmount"`
	PayableAmount       string `xml:"PayableAmount"`
}

// --- Invoice lines ---

type xmlInvoiceLine struct {
	ID                  string        `xml:"ID"`
	InvoicedQuantity    string        `xml:"InvoicedQuantity"`
	LineExtensionAmount string        `xml:"LineExtensionAmount"`
	TaxTotal            []xmlTaxTotal `xml:"TaxTotal"`
	Item                xmlItem       `xml:"Item"`
	Price               xmlPrice      `xml:"Price"`
}

type xmlItem struct {
	// []string, not string: some invoices have a duplicated Description
	// node inside the same Item; only the first one is used.
	Description                []string                      `xml:"Description"`
	StandardItemIdentification xmlStandardItemIdentification `xml:"StandardItemIdentification"`
}

type xmlStandardItemIdentification struct {
	ID string `xml:"ID"` // can be a self-closed empty tag
}

type xmlPrice struct {
	PriceAmount string `xml:"PriceAmount"`
}
