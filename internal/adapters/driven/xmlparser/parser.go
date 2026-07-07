package xmlparser

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"cifrato/internal/application/ports/out"
	"cifrato/internal/domain/invoice"
)

type Parser struct{}

func NewParser() *Parser { return &Parser{} }

var _ out.InvoiceParser = (*Parser)(nil)

func (p *Parser) Parse(xmlData []byte) (*invoice.Invoice, error) {
	root, err := detectRootElement(xmlData)
	if err != nil {
		return nil, fmt.Errorf("xmlparser: detecting root element: %w", err)
	}

	switch root {
	case "Invoice":
		var xi xmlInvoice
		if err := xml.Unmarshal(xmlData, &xi); err != nil {
			return nil, fmt.Errorf("xmlparser: unmarshaling Invoice: %w", err)
		}
		inv, err := mapInvoice(&xi, invoice.XMLTypeInvoice)
		if err != nil {
			return nil, fmt.Errorf("xmlparser: %w", err)
		}
		return inv, nil

	case "AttachedDocument":
		var ad xmlAttachedDocument
		if err := xml.Unmarshal(xmlData, &ad); err != nil {
			return nil, fmt.Errorf("xmlparser: unmarshaling AttachedDocument: %w", err)
		}
		embedded := []byte(ad.Attachment.ExternalReference.Description)
		if len(bytes.TrimSpace(embedded)) == 0 {
			return nil, fmt.Errorf("xmlparser: AttachedDocument has no embedded Invoice in Attachment/ExternalReference/Description")
		}
		var xi xmlInvoice
		if err := xml.Unmarshal(embedded, &xi); err != nil {
			return nil, fmt.Errorf("xmlparser: unmarshaling embedded Invoice from AttachedDocument: %w", err)
		}
		inv, err := mapInvoice(&xi, invoice.XMLTypeAttachedDocument)
		if err != nil {
			return nil, fmt.Errorf("xmlparser: %w", err)
		}
		return inv, nil

	default:
		return nil, fmt.Errorf("xmlparser: unrecognized root element %q (expected Invoice or AttachedDocument)", root)
	}
}

// detectRootElement peeks the document with a streaming xml.Decoder and
// returns the local name of the first StartElement, without unmarshaling
// the whole document twice.
func detectRootElement(xmlData []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("scanning for root element: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}
