package store

import (
	"context"
	"fmt"
	"strings"
)

// FinishExecution releases only the request slot. The financial hold and token
// estimates continue to protect the quota until Settle or Release completes.
// A repeated notification, including one after settlement, is harmless.
func (s *Store) FinishExecution(ctx context.Context, reservationID string) error {
	if strings.TrimSpace(reservationID) == "" {
		return fmt.Errorf("%w: reservation id is required", ErrInvalidArgument)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE reservations
		SET local_execution_finished_at_unix_ms = ?
		WHERE id = ? AND status = 'held' AND local_execution_finished_at_unix_ms IS NULL`, nowUnixMilli(), reservationID)
	if err != nil {
		return fmt.Errorf("finish request execution: %w", err)
	}
	return nil
}
