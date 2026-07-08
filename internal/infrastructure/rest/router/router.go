package router

import (
	"log"
	"net/http"

	"go.uber.org/dig"

	"cifrato/internal/infrastructure/rest/handlers"
)

// NewRouter wires the HTTP driving adapter's routes. Uses the stdlib
// ServeMux pattern matching (Go 1.22+) — no external router dependency,
// consistent with the project's minimal-dependency approach. Resolves its
// handlers from the dig container at registration time, the same pattern
// bia-electronic-bills' router.NewRouter(container) uses.
func NewRouter(container *dig.Container) *http.ServeMux {
	mux := http.NewServeMux()
	err := container.Invoke(func(invoiceHandler *handlers.InvoiceHandler) {
		mux.HandleFunc("POST /invoices", invoiceHandler.HandleProcess)
	})
	if err != nil {
		log.Fatal(err)
	}
	return mux
}
