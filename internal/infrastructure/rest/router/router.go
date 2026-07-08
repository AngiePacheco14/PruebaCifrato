package router

import (
	"log"
	"net/http"

	"go.uber.org/dig"

	"cifrato/internal/infrastructure/rest/handlers"
)

// NewRouter wires the HTTP driving adapter's routes using stdlib ServeMux
// pattern matching, resolving handlers from the dig container.
func NewRouter(container *dig.Container) *http.ServeMux {
	mux := http.NewServeMux()
	err := container.Invoke(func(invoiceHandler *handlers.InvoiceHandler) {
		mux.HandleFunc("POST /invoices", invoiceHandler.HandleProcess)
		mux.HandleFunc("POST /invoices/batch", invoiceHandler.HandleProcessBatch)
	})
	if err != nil {
		log.Fatal(err)
	}
	return mux
}
