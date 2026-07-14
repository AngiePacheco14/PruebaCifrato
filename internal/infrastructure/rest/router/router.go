package router

import (
	"log"
	"net/http"

	"go.uber.org/dig"

	"cifrato/internal/infrastructure/rest/handlers"
	"cifrato/internal/infrastructure/rest/middleware"
)

// NewRouter wires the HTTP driving adapter's routes using stdlib ServeMux
// pattern matching, resolving handlers from the dig container, and wraps the
// result with the CORS middleware.
func NewRouter(container *dig.Container) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	err := container.Invoke(func(invoiceHandler *handlers.InvoiceHandler) {
		mux.HandleFunc("POST /invoices", invoiceHandler.HandleProcess)
		mux.HandleFunc("POST /invoices/batch", invoiceHandler.HandleProcessBatch)
	})
	if err != nil {
		log.Fatal(err)
	}
	return middleware.CORS(mux)
}

// handleHealth is a cheap liveness check for hosting platforms (e.g. Render)
// that need a GET endpoint to confirm the service is up.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
