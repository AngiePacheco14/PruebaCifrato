package xmlparser

import (
	"os"
	"testing"

	"github.com/shopspring/decimal"

	"cifrato/internal/domain/invoice"
)

const fixturesDir = "../../../../sample-invoices"

func decStr(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("bad literal %q: %v", s, err)
	}
	return d
}

type tc struct {
	name           string
	path           string
	wantXMLType    invoice.XMLType
	wantCUFE       string
	wantNumber     string
	wantIssueDate  string // "2006-01-02"
	wantIssuerNIT  string
	wantBuyerNIT   string
	wantIssuerName string
	wantSubtotal   string
	wantIVATotal   string
	wantTotal      string
	wantLineCount  int
	extra          func(t *testing.T, inv *invoice.Invoice)
}

func TestParse_RealInvoices(t *testing.T) {
	p := NewParser()

	cases := []tc{
		{
			name:          "FLA115451 AttachedDocument wraps an Invoice, ignores the nested ApplicationResponse",
			path:          fixturesDir + "/sample-1/2025-08-03_FLA115451_e64ed6fd1495fc84.xml",
			wantXMLType:   invoice.XMLTypeAttachedDocument,
			wantIssueDate: "2025-08-03",
			wantIssuerNIT: "900798160",
			wantLineCount: 4,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if len(inv.Lines) == 0 {
					t.Fatal("expected at least 1 line")
				}
				if inv.ReportedRetefuente != nil || inv.ReportedReteiva != nil || inv.ReportedReteica != nil {
					t.Errorf("expected no withholding reported, got %+v/%+v/%+v", inv.ReportedRetefuente, inv.ReportedReteiva, inv.ReportedReteica)
				}
			},
		},
		{
			name:          "FEAM168587 self-closed StandardItemIdentification ID yields nil SKU",
			path:          fixturesDir + "/sample-1/2025-08-04_FEAM168587_719ce8fd88c792c1.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "FEAM168587",
			wantIssueDate: "2025-08-04",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.Lines[0].SKU != nil {
					t.Errorf("SKU = %v, want nil (self-closed <cbc:ID schemeID=\"999\" />)", *inv.Lines[0].SKU)
				}
				if inv.ReportedRetefuente != nil || inv.ReportedReteiva != nil || inv.ReportedReteica != nil {
					t.Error("expected no WithholdingTaxTotal at all")
				}
			},
		},
		{
			name:          "P12206 duplicate IVA TaxTotal block deduplicated before summing",
			path:          fixturesDir + "/sample-1/2025-09-05_P12206_499c1c1fd58b39f6.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "P12206",
			wantIssueDate: "2025-09-05",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				// Subtotal + IVA must reconcile to InvoiceTotal once deduplicated.
				sum := inv.Subtotal.Add(inv.IVATotal)
				if !sum.Equal(inv.InvoiceTotal) {
					t.Errorf("Subtotal(%s) + IVATotal(%s) = %s, want InvoiceTotal %s", inv.Subtotal, inv.IVATotal, sum, inv.InvoiceTotal)
				}
			},
		},
		{
			name:          "MRBA27913 WithholdingTaxTotal with unrecognized counterparty names is ignored",
			path:          fixturesDir + "/sample-1/2025-12-17_MRBA27913_bde00ee01ae83f48.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "MRBA27913",
			wantIssueDate: "2025-12-17",
			wantLineCount: 6,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.ReportedRetefuente != nil || inv.ReportedReteiva != nil || inv.ReportedReteica != nil {
					t.Errorf("expected all withholding fields nil, got %+v/%+v/%+v", inv.ReportedRetefuente, inv.ReportedReteiva, inv.ReportedReteica)
				}
			},
		},
		{
			name:          "FES23029 ReteRenta + ReteICA classification via flexible name matching",
			path:          fixturesDir + "/sample-1/2026-04-22_FES23029_4ea525948bd872b9.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "FES23029",
			wantIssueDate: "2026-04-22",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.ReportedRetefuente == nil {
					t.Error("ReportedRetefuente = nil, want a value classified from Name containing \"ReteRenta\"")
				}
				if inv.ReportedReteica == nil {
					t.Error("ReportedReteica = nil, want a value")
				}
				if inv.ReportedReteiva != nil {
					t.Errorf("ReportedReteiva = %v, want nil", inv.ReportedReteiva)
				}
			},
		},
		{
			name:          "POST6081 duplicate Description nodes, first one wins",
			path:          fixturesDir + "/sample-2/2026-03-02_POST6081_721c2363e93b2a31.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "POST6081",
			wantIssueDate: "2026-03-02",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.Lines[0].Description != "CONVERTIDOR" {
					t.Errorf("Description = %q, want CONVERTIDOR (deduplicated)", inv.Lines[0].Description)
				}
			},
		},
		{
			name:          "FEPR3143556 no header TaxTotal at all, IVATotal zero, no withholding",
			path:          fixturesDir + "/sample-2/2026-03-08_FEPR3143556_c36e80d488987cdf.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "FEPR3143556",
			wantIssueDate: "2026-03-08",
			wantIVATotal:  "0",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.ReportedRetefuente != nil || inv.ReportedReteiva != nil || inv.ReportedReteica != nil {
					t.Error("expected no withholding fields set")
				}
			},
		},
		{
			name:          "POST6301 duplicate Description nodes, first one wins (second instance)",
			path:          fixturesDir + "/sample-2/2026-03-18_POST6301_98c7cff0264cc51a.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "POST6301",
			wantIssueDate: "2026-03-18",
			wantLineCount: 1,
		},
		{
			name:          "FE03423 multiple header TaxTotal incl. ICUI excluded, ZY withholding with empty Name ignored",
			path:          fixturesDir + "/sample-2/2026-04-07_FE03423_6f704bdcf9b681f6.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "FE03423",
			wantIssueDate: "2026-04-07",
			wantLineCount: 11,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.ReportedRetefuente == nil {
					t.Error("ReportedRetefuente = nil, want a value classified from Name containing \"ReteRenta\"")
				}
				if inv.ReportedReteiva != nil {
					t.Errorf("ReportedReteiva = %v, want nil (ZY block has empty Name, ignored)", inv.ReportedReteiva)
				}
				if inv.ReportedReteica != nil {
					t.Errorf("ReportedReteica = %v, want nil", inv.ReportedReteica)
				}
			},
		},
		{
			name:          `"59" no header TaxTotal, no real WithholdingTaxTotal`,
			path:          fixturesDir + "/sample-2/2026-04-16_59_5109232f07a260e8.xml",
			wantXMLType:   invoice.XMLTypeInvoice,
			wantNumber:    "59",
			wantIssueDate: "2026-04-16",
			wantIVATotal:  "0",
			wantLineCount: 1,
			extra: func(t *testing.T, inv *invoice.Invoice) {
				if inv.ReportedRetefuente != nil || inv.ReportedReteiva != nil || inv.ReportedReteica != nil {
					t.Error("expected no withholding fields set")
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := os.ReadFile(c.path)
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}

			inv, err := p.Parse(data)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if inv.XMLType != c.wantXMLType {
				t.Errorf("XMLType = %v, want %v", inv.XMLType, c.wantXMLType)
			}
			if c.wantCUFE != "" && inv.CUFE != c.wantCUFE {
				t.Errorf("CUFE = %q, want %q", inv.CUFE, c.wantCUFE)
			}
			if inv.CUFE == "" {
				t.Error("CUFE is empty, want a value")
			}
			if c.wantNumber != "" && inv.InvoiceNumber != c.wantNumber {
				t.Errorf("InvoiceNumber = %q, want %q", inv.InvoiceNumber, c.wantNumber)
			}
			if inv.IssueDate.Format("2006-01-02") != c.wantIssueDate {
				t.Errorf("IssueDate = %v, want %v", inv.IssueDate, c.wantIssueDate)
			}
			if c.wantIssuerNIT != "" && inv.IssuerNIT != c.wantIssuerNIT {
				t.Errorf("IssuerNIT = %q, want %q", inv.IssuerNIT, c.wantIssuerNIT)
			}
			if inv.IssuerNIT == "" {
				t.Error("IssuerNIT is empty, want a value")
			}
			if inv.BuyerNIT == "" {
				t.Error("BuyerNIT is empty, want a value")
			}
			if c.wantIssuerName != "" && inv.IssuerName != c.wantIssuerName {
				t.Errorf("IssuerName = %q, want %q", inv.IssuerName, c.wantIssuerName)
			}
			if c.wantIVATotal != "" && !inv.IVATotal.Equal(decStr(t, c.wantIVATotal)) {
				t.Errorf("IVATotal = %s, want %s", inv.IVATotal, c.wantIVATotal)
			}
			if inv.InvoiceTotal.IsZero() {
				t.Error("InvoiceTotal is zero, want a positive value")
			}
			if len(inv.Lines) != c.wantLineCount {
				t.Errorf("len(Lines) = %d, want %d", len(inv.Lines), c.wantLineCount)
			}
			if inv.SourceXMLPath != "" || inv.SourcePDFPath != "" {
				t.Error("Parse must not set SourceXMLPath/SourcePDFPath; that's the caller's job")
			}

			if c.extra != nil {
				c.extra(t, inv)
			}
		})
	}
}

func TestParse_MalformedXMLReturnsError(t *testing.T) {
	p := NewParser()
	if _, err := p.Parse([]byte("not xml at all")); err == nil {
		t.Fatal("expected an error for malformed XML input")
	}
}

func TestParse_UnknownRootElementReturnsError(t *testing.T) {
	p := NewParser()
	if _, err := p.Parse([]byte(`<?xml version="1.0"?><SomethingElse/>`)); err == nil {
		t.Fatal("expected an error for an unrecognized root element")
	}
}

func TestParse_MissingRequiredFieldReturnsError(t *testing.T) {
	p := NewParser()
	xmlWithoutCUFE := []byte(`<?xml version="1.0"?><Invoice><ID>X</ID></Invoice>`)
	if _, err := p.Parse(xmlWithoutCUFE); err == nil {
		t.Fatal("expected an error when CUFE (UUID) is missing")
	}
}
