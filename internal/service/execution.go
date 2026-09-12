package service

import (
	"context"
	"time"
)

// FinishExecution is independent of billing. Keep authPending for delayed usage
// callbacks, but remove it from the selected credential's execution count.
func (s *Service) FinishExecution(ctx context.Context, reservationID string) error {
	// The client may already have cancelled its request. Cleanup still needs a
	// live, bounded context in order to return its slot.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.store.FinishExecution(cleanupCtx, reservationID); err != nil {
		return err
	}
	s.FinishAuthCapture(reservationID)
	return nil
}
