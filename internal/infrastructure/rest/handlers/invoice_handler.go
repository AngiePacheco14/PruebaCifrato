package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sync"

	"cifrato/internal/application/ports/in"
	"cifrato/internal/infrastructure/rest/dto"
)

// maxUploadBytes bounds how much of the request body is read before giving up.
const maxUploadBytes = 10 << 20 // 10 MiB

// maxBatchConcurrency caps concurrent invoice processing in HandleProcessBatch,
// kept below the DB pool's max open connections to avoid starving it.
const maxBatchConcurrency = 5

type InvoiceHandler struct {
	processInvoice in.ProcessInvoice
}

func NewInvoiceHandler(processInvoice in.ProcessInvoice) *InvoiceHandler {
	return &InvoiceHandler{processInvoice: processInvoice}
}

// HandleProcess implements POST /invoices: body is raw UBL DIAN XML, runs the
// full parse → save → classify → calculate pipeline, returns JSON results.
func (h *InvoiceHandler) HandleProcess(w http.ResponseWriter, r *http.Request) {
	body, err := readLimited(r.Body)
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

// HandleProcessBatch implements POST /invoices/batch: multipart files under
// "files" are each run through HandleProcess's pipeline independently. A
// failed file just reports success=false; the response is still 200.
func (h *InvoiceHandler) HandleProcessBatch(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "parsing multipart form: "+err.Error())
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, `no files provided under the "files" field`)
		return
	}

	results := make([]dto.ProcessInvoiceBatchItemDTO, len(files))
	sem := make(chan struct{}, maxBatchConcurrency)
	var wg sync.WaitGroup
	for i, fh := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, fh *multipart.FileHeader) {
			defer wg.Done()
			defer func() { <-sem }()
			// Each goroutine writes to its own index — safe without a mutex,
			// results is never resized after this point.
			results[i] = h.processBatchFile(r.Context(), fh)
		}(i, fh)
	}
	wg.Wait()

	writeJSON(w, http.StatusOK, dto.ProcessInvoicesBatchResponse{Results: results})
}

func (h *InvoiceHandler) processBatchFile(ctx context.Context, fh *multipart.FileHeader) dto.ProcessInvoiceBatchItemDTO {
	item := dto.ProcessInvoiceBatchItemDTO{Filename: fh.Filename}

	f, err := fh.Open()
	if err != nil {
		item.Error = "opening file: " + err.Error()
		return item
	}
	defer f.Close()

	body, err := readLimited(f)
	if err != nil {
		item.Error = "reading file: " + err.Error()
		return item
	}
	if len(body) > maxUploadBytes {
		item.Error = "file exceeds maximum allowed size"
		return item
	}

	result, err := h.processInvoice.Execute(ctx, body, fh.Filename, "")
	if err != nil {
		log.Printf("invoice_handler: batch item %q failed: %v", fh.Filename, err)
		item.Error = err.Error()
		return item
	}

	response := toProcessInvoiceResponse(result)
	item.Success = true
	item.Invoice = &response
	return item
}

// readLimited reads up to maxUploadBytes+1 bytes from r, so callers can
// detect an oversized body by checking len(body) against maxUploadBytes.
func readLimited(r io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxUploadBytes+1))
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
