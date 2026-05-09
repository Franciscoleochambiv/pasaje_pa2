package service

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeatChangeNotifier is the interface for broadcasting seat updates.
// This avoids a circular import with the ws package.
type SeatChangeNotifier interface {
	NotifySeatChange(ctx context.Context, tripInstanceID int64)
	NotifyPaymentStatusChange(reservationCode string, status string, rejectionReason string, extra map[string]any)
}

// ExpiryService libera asientos held cuya reserva ha expirado.
type ExpiryService struct {
	Pool     *pgxpool.Pool
	Notifier SeatChangeNotifier // can be nil
}

// ReleaseExpiredHolds libera asientos held vencidos, marca sus reservas como expired
// y elimina los reservation_items para permitir re-reserva.
// Returns the number of released seats and the affected trip_instance_ids.
func (s *ExpiryService) ReleaseExpiredHolds(ctx context.Context) (int64, []int64, error) {
	now := time.Now()

	// Use a transaction to keep seat inventory, reservations and items consistent.
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback(ctx)

	// 0. Query affected trip_instance_ids BEFORE releasing
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT trip_instance_id FROM trip_seat_inventory
		WHERE status = 'held' AND hold_expires_at IS NOT NULL AND hold_expires_at < $1
	`, now)
	if err != nil {
		return 0, nil, err
	}
	var affectedTrips []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, nil, err
		}
		affectedTrips = append(affectedTrips, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	// 1. Liberar asientos held cuyo hold_expires_at ya pasó
	tag, err := tx.Exec(ctx, `
		UPDATE trip_seat_inventory
		SET status = 'available', hold_expires_at = NULL, held_by = NULL, updated_at = now()
		WHERE status = 'held' AND hold_expires_at IS NOT NULL AND hold_expires_at < $1
	`, now)
	if err != nil {
		return 0, nil, err
	}
	released := tag.RowsAffected()

	// 2. Find pending_verification reservations that are about to expire (for WS notification)
	var expiredVerificationCodes []string
	pvRows, err := tx.Query(ctx, `
		SELECT code FROM reservations
		WHERE status = 'pending_verification' AND expires_at < $1
	`, now)
	if err == nil {
		for pvRows.Next() {
			var code string
			if pvRows.Scan(&code) == nil {
				expiredVerificationCodes = append(expiredVerificationCodes, code)
			}
		}
		pvRows.Close()
	}

	// 3. Marcar reservas pending y pending_verification expiradas
	_, err = tx.Exec(ctx, `
		UPDATE reservations
		SET status = 'expired', updated_at = now()
		WHERE status IN ('pending', 'pending_verification') AND expires_at < $1
	`, now)
	if err != nil {
		return released, affectedTrips, err
	}

	// 3b. Eliminar reservation_items de reservas expiradas para permitir re-reserva
	_, err = tx.Exec(ctx, `
		DELETE FROM reservation_items
		WHERE reservation_id IN (
			SELECT id FROM reservations
			WHERE status = 'expired' AND expires_at < $1
		)
	`, now)
	if err != nil {
		return released, affectedTrips, err
	}

	// 4. Mark associated Yape direct payments as expired too
	_, _ = tx.Exec(ctx, `
		UPDATE payments SET status = 'expired', updated_at = now()
		WHERE status = 'pending_verification'
		AND reservation_id IN (SELECT id FROM reservations WHERE status = 'expired')
	`)

	if err := tx.Commit(ctx); err != nil {
		return released, affectedTrips, err
	}

	// 5. Notify customers whose pending_verification reservations expired
	if s.Notifier != nil {
		for _, code := range expiredVerificationCodes {
			s.Notifier.NotifyPaymentStatusChange(code, "expired", "Tiempo de verificacion expirado", nil)
		}
	}

	return released, affectedTrips, nil
}

// StartExpiryWorker inicia un goroutine que cada intervalo revisa y libera holds expirados.
func (s *ExpiryService) StartExpiryWorker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("expiry worker: running every %s", interval)

		for {
			select {
			case <-ctx.Done():
				log.Println("expiry worker: stopped")
				return
			case <-ticker.C:
				released, affectedTrips, err := s.ReleaseExpiredHolds(ctx)
				if err != nil {
					log.Printf("expiry worker: error releasing holds: %v", err)
				} else if released > 0 {
					log.Printf("expiry worker: released %d expired seat holds", released)
					// Broadcast seat updates for affected trips
					if s.Notifier != nil {
						for _, tripID := range affectedTrips {
							s.Notifier.NotifySeatChange(ctx, tripID)
						}
					}
				}
			}
		}
	}()
}
