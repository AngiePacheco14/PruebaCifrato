package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"cifrato/internal/application/ports/in"
	"cifrato/internal/infrastructure/rest/dto"
)

// maxUploadBytes bounds how much of the request body is read before
// giving up — UBL DIAN invoices are XML documents in the tens/hundreds of
// KB range, not a streaming media use case.
const maxUploadBytes = 10 << 20 // 10 MiB

type InvoiceHandler struct {
	processInvoice in.ProcessInvoice
}

func NewInvoiceHandler(processInvoice in.ProcessInvoice) *InvoiceHandler {
	return &InvoiceHandler{processInvoice: processInvoice}
}

// HandleProcess implements POST /invoices: the request body is the raw
// UBL DIAN invoice XML (Content-Type: application/xml). It runs the full
// parse → save → classify → calculate pipeline and returns the resulting
// withholding calculations as JSON.
func (h *InvoiceHandler) HandleProcess(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "reading request body: "+err.Error())
		return
	}
	if len(body) == 0 {
		writeError(w, http.StatusBadRequest, "request body is empty")
		return
	}
	if len(body) > maxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds maximum allowed size")
		return
	}

	sourceXMLPath := r.URL.Query().Get("filename")

	result, err := h.processInvoice.Execute(r.Context(), body, sourceXMLPath, "")
	if err != nil {
		log.Printf("invoice_handler: processing invoice failed: %v", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toProcessInvoiceResponse(result))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("invoice_handler: encoding response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
