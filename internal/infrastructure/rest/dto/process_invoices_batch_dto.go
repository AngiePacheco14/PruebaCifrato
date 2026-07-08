package dto

// ProcessInvoicesBatchResponse is the JSON body returned by POST
// /invoices/batch — one result per uploaded file, in the order received.
type ProcessInvoicesBatchResponse struct {
	Results []ProcessInvoiceBatchItemDTO `json:"results"`
}

// ProcessInvoiceBatchItemDTO reports one file's outcome; a failed file only
// sets Success=false and Error for itself, without failing the batch.
type ProcessInvoiceBatchItemDTO struct {
	Filename string                  `json:"filename"`
	Success  bool                    `json:"success"`
	Invoice  *ProcessInvoiceResponse `json:"invoice,omitempty"`
	Error    string                  `json:"error,omitempty"`
}
