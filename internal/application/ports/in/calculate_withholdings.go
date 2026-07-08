package in

import (
	"context"

	"cifrato/internal/domain/entity"
)

// CalculateWithholdings computes and persists RETEFUENTE/RETEIVA/RETEICA
// for every line of an already-persisted invoice (inv.ID and each
// inv.Lines[i].ID must be set — this runs after InvoiceRepository.Save).
// Implemented by usecase.CalculateWithholdings.
type CalculateWithholdings interface {
	Execute(ctx context.Context, inv *entity.Invoice) ([]entity.Calculation, error)
}
