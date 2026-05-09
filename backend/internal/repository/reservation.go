package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReservationRepository acceso a datos de reservas, tickets y pagos.
type ReservationRepository struct {
	Pool *pgxpool.Pool
}

// GetOrCreateWebChannel devuelve el ID del canal "web", creándolo si no existe.
func (r *ReservationRepository) GetOrCreateWebChannel(ctx context.Context) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `SELECT id FROM sales_channels WHERE code = 'web'`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.Pool.QueryRow(ctx, `
			INSERT INTO sales_channels (name, code) VALUES ('Web', 'web')
			ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
			RETURNING id
		`).Scan(&id)
	}
	return id, err
}

// GetOrCreatePOSChannel devuelve el ID del canal "pos", creándolo si no existe.
func (r *ReservationRepository) GetOrCreatePOSChannel(ctx context.Context) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `SELECT id FROM sales_channels WHERE code = 'pos'`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.Pool.QueryRow(ctx, `
			INSERT INTO sales_channels (name, code) VALUES ('Punto de Venta', 'pos')
			ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
			RETURNING id
		`).Scan(&id)
	}
	return id, err
}

// CreateReservation crea una reserva con sus ítems en una transacción.
// Bloquea los asientos con SELECT ... FOR UPDATE, verifica disponibilidad,
// actualiza a held, crea la reserva y los ítems.
func (r *ReservationRepository) CreateReservation(
	ctx context.Context,
	code string,
	tripInstanceID int64,
	seatIDs []int64,
	salesChannelID int64,
	expiresAt time.Time,
	contactName string,
	contactDocNumber string,
	contactPhone string,
) (*domain.Reservation, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Bloquear y verificar asientos
	rows, err := tx.Query(ctx, `
		SELECT id, status FROM trip_seat_inventory
		WHERE id = ANY($1) AND trip_instance_id = $2
		FOR UPDATE
	`, seatIDs, tripInstanceID)
	if err != nil {
		return nil, fmt.Errorf("lock seats: %w", err)
	}

	found := make(map[int64]string)
	for rows.Next() {
		var id int64
		var status string
		if err := rows.Scan(&id, &status); err != nil {
			rows.Close()
			return nil, err
		}
		found[id] = status
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if len(found) != len(seatIDs) {
		return nil, fmt.Errorf("uno o más asientos no encontrados para este viaje")
	}
	for _, sid := range seatIDs {
		if found[sid] != "available" {
			return nil, fmt.Errorf("el asiento %d no está disponible (estado: %s)", sid, found[sid])
		}
	}

	// Actualizar asientos a held
	_, err = tx.Exec(ctx, `
		UPDATE trip_seat_inventory
		SET status = 'held', hold_expires_at = $2, held_by = $3, updated_at = now()
		WHERE id = ANY($1)
	`, seatIDs, expiresAt, code)
	if err != nil {
		return nil, fmt.Errorf("hold seats: %w", err)
	}

	// Crear reserva
	var res domain.Reservation
	err = tx.QueryRow(ctx, `
		INSERT INTO reservations (code, sales_channel_id, status, expires_at, contact_name, contact_doc_number, contact_phone)
		VALUES ($1, $2, 'pending', $3, $4, $5, $6)
		RETURNING id, code, sales_channel_id, status, expires_at, created_at, updated_at, contact_name, contact_doc_number, contact_phone
	`, code, salesChannelID, expiresAt, contactName, contactDocNumber, contactPhone).Scan(
		&res.ID, &res.Code, &res.SalesChannelID, &res.Status, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt,
		&res.ContactName, &res.ContactDocNumber, &res.ContactPhone,
	)
	if err != nil {
		return nil, fmt.Errorf("insert reservation: %w", err)
	}

	// Crear ítems
	for _, sid := range seatIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO reservation_items (reservation_id, trip_seat_inventory_id) VALUES ($1, $2)
		`, res.ID, sid)
		if err != nil {
			return nil, fmt.Errorf("insert reservation item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &res, nil
}

// ConfirmData agrupa los datos necesarios para confirmar una reserva.
type ConfirmData struct {
	PaymentMethod      string
	PaymentReference   string
	PassengerName      string
	PassengerDocType   string
	PassengerDocNumber string
	PassengerEmail     string
	PassengerPhone     string
	PassengerAddress   string
	DocumentType       string
	RouteName          string
	SeatLabels         []string
	PricePerSeat       float64
}

// ConfirmReservation confirma una reserva: actualiza asientos a sold, crea tickets y pago.
func (r *ReservationRepository) ConfirmReservation(
	ctx context.Context,
	code string,
	input *ConfirmData,
) ([]string, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Obtener la reserva con lock
	var res domain.Reservation
	err = tx.QueryRow(ctx, `
		SELECT id, code, sales_channel_id, status, expires_at
		FROM reservations WHERE code = $1 FOR UPDATE
	`, code).Scan(&res.ID, &res.Code, &res.SalesChannelID, &res.Status, &res.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("reserva no encontrada")
		}
		return nil, err
	}
	if res.Status != "pending" && res.Status != "pending_verification" {
		return nil, fmt.Errorf("la reserva no está pendiente (estado: %s)", res.Status)
	}
	if time.Now().After(res.ExpiresAt) {
		return nil, fmt.Errorf("la reserva ha expirado")
	}

	// Obtener ítems con sus seat inventory + trip_instance_id
	type itemInfo struct {
		seatInventoryID int64
		tripInstanceID  int64
	}
	irows, err := tx.Query(ctx, `
		SELECT ri.trip_seat_inventory_id, tsi.trip_instance_id
		FROM reservation_items ri
		JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		WHERE ri.reservation_id = $1
	`, res.ID)
	if err != nil {
		return nil, err
	}

	var items []itemInfo
	for irows.Next() {
		var it itemInfo
		if err := irows.Scan(&it.seatInventoryID, &it.tripInstanceID); err != nil {
			irows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	if err := irows.Err(); err != nil {
		irows.Close()
		return nil, err
	}
	irows.Close()

	// Actualizar asientos a sold
	seatIDs := make([]int64, len(items))
	for i, it := range items {
		seatIDs[i] = it.seatInventoryID
	}
	_, err = tx.Exec(ctx, `
		UPDATE trip_seat_inventory
		SET status = 'sold', hold_expires_at = NULL, updated_at = now()
		WHERE id = ANY($1)
	`, seatIDs)
	if err != nil {
		return nil, fmt.Errorf("update seats to sold: %w", err)
	}

	// Actualizar reserva a confirmed
	_, err = tx.Exec(ctx, `
		UPDATE reservations SET status = 'confirmed', updated_at = now() WHERE id = $1
	`, res.ID)
	if err != nil {
		return nil, fmt.Errorf("confirm reservation: %w", err)
	}

	// Crear tickets y pagos
	var ticketCodes []string
	for _, it := range items {
		ticketCode := generateTicketCode()
		var ticketID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO tickets (code, reservation_id, trip_instance_id, trip_seat_inventory_id, sales_channel_id, status)
			VALUES ($1, $2, $3, $4, $5, 'issued')
			RETURNING id
		`, ticketCode, res.ID, it.tripInstanceID, it.seatInventoryID, res.SalesChannelID).Scan(&ticketID)
		if err != nil {
			return nil, fmt.Errorf("insert ticket: %w", err)
		}

		var ref *string
		if input.PaymentReference != "" {
			ref = &input.PaymentReference
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO payments (
				ticket_id, reservation_id, amount_cents, currency, method, reference, status,
				passenger_name, passenger_doc_type, passenger_doc_number, passenger_email,
				passenger_phone, passenger_address, document_type, route_name, seat_labels,
				trip_instance_id, price_per_seat
			) VALUES ($1, $2, 0, 'PEN', $3, $4, 'completed', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`, ticketID, res.ID, input.PaymentMethod, ref,
			input.PassengerName, input.PassengerDocType, input.PassengerDocNumber,
			input.PassengerEmail, input.PassengerPhone, input.PassengerAddress,
			input.DocumentType, input.RouteName, input.SeatLabels,
			it.tripInstanceID, input.PricePerSeat)
		if err != nil {
			return nil, fmt.Errorf("insert payment: %w", err)
		}

		ticketCodes = append(ticketCodes, ticketCode)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return ticketCodes, nil
}

// GetByID devuelve una reserva por su ID.
func (r *ReservationRepository) GetByID(ctx context.Context, id int64) (*domain.Reservation, error) {
	var res domain.Reservation
	err := r.Pool.QueryRow(ctx, `
		SELECT id, code, user_id, agency_id, sales_channel_id, status, expires_at, created_at, updated_at, contact_name, contact_doc_number, contact_phone
		FROM reservations WHERE id = $1
	`, id).Scan(&res.ID, &res.Code, &res.UserID, &res.AgencyID, &res.SalesChannelID, &res.Status, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt, &res.ContactName, &res.ContactDocNumber, &res.ContactPhone)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetByCode devuelve una reserva por su código.
func (r *ReservationRepository) GetByCode(ctx context.Context, code string) (*domain.Reservation, error) {
	var res domain.Reservation
	err := r.Pool.QueryRow(ctx, `
		SELECT id, code, user_id, agency_id, sales_channel_id, status, expires_at, created_at, updated_at, contact_name, contact_doc_number, contact_phone
		FROM reservations WHERE code = $1
	`, code).Scan(&res.ID, &res.Code, &res.UserID, &res.AgencyID, &res.SalesChannelID, &res.Status, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt, &res.ContactName, &res.ContactDocNumber, &res.ContactPhone)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// PendingReservationView es una reserva pending con sus asientos para el panel admin.
type PendingReservationView struct {
	ID               int64     `json:"id"`
	Code             string    `json:"code"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	ContactName      string    `json:"contact_name"`
	ContactDocNumber string    `json:"contact_doc_number"`
	ContactPhone     string    `json:"contact_phone"`
	SeatLabels       []string  `json:"seat_labels"`
}

// ListPendingReservationsByTrip devuelve las reservas pending de un viaje con sus asientos.
func (r *ReservationRepository) ListPendingReservationsByTrip(ctx context.Context, tripInstanceID int64) ([]PendingReservationView, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT res.id, res.code, res.status, res.expires_at, res.created_at,
		       COALESCE(res.contact_name, ''), COALESCE(res.contact_doc_number, ''), COALESCE(res.contact_phone, ''),
		       array_agg(vs.label ORDER BY vs.position)
		FROM reservations res
		JOIN reservation_items ri ON ri.reservation_id = res.id
		JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		JOIN vehicle_seats vs ON vs.id = tsi.vehicle_seat_id
		WHERE tsi.trip_instance_id = $1 AND res.status = 'pending'
		GROUP BY res.id, res.code, res.status, res.expires_at, res.created_at, res.contact_doc_number, res.contact_phone
		ORDER BY res.created_at DESC
	`, tripInstanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PendingReservationView
	for rows.Next() {
		var v PendingReservationView
		var labels []string
		if err := rows.Scan(&v.ID, &v.Code, &v.Status, &v.ExpiresAt, &v.CreatedAt,
			&v.ContactName, &v.ContactDocNumber, &v.ContactPhone, &labels); err != nil {
			return nil, err
		}
		v.SeatLabels = labels
		results = append(results, v)
	}
	return results, rows.Err()
}

// GetPublicByCode devuelve la vista enriquecida de una reserva: datos del
// pasajero (de payments), asientos con etiqueta + ticket asociado, y datos
// del viaje (ruta, paraderos, fecha de salida). Es lo que consume la consulta
// pública del cliente en web/mobile.
func (r *ReservationRepository) GetPublicByCode(ctx context.Context, code string) (*domain.ReservationPublicInfo, error) {
	var info domain.ReservationPublicInfo
	var billingSaleID *int
	var tripInstanceID *int64
	var departureAt *time.Time
	var amountCents *int64

	err := r.Pool.QueryRow(ctx, `
		SELECT res.code,
		       res.status,
		       res.expires_at,
		       res.created_at,
		       COALESCE(p.passenger_name, ''),
		       COALESCE(p.passenger_doc_type, ''),
		       COALESCE(p.passenger_doc_number, ''),
		       COALESCE(p.passenger_email, ''),
		       COALESCE(p.passenger_phone, ''),
		       COALESCE(p.document_type, ''),
		       COALESCE(p.method, ''),
		       COALESCE(p.status, ''),
		       p.billing_sale_id,
		       p.amount_cents,
		       COALESCE(p.route_name, ''),
		       p.trip_instance_id,
		       ti.departure_at,
		       COALESCE(orig.name, ''),
		       COALESCE(dest.name, '')
		FROM reservations res
		LEFT JOIN LATERAL (
		    SELECT * FROM payments WHERE reservation_id = res.id
		     ORDER BY created_at DESC LIMIT 1
		) p ON TRUE
		LEFT JOIN trip_instances ti ON ti.id = p.trip_instance_id
		LEFT JOIN trip_templates tt ON tt.id = ti.trip_template_id
		LEFT JOIN routes ro ON ro.id = tt.route_id
		LEFT JOIN LATERAL (
		    SELECT name FROM stops WHERE route_id = ro.id
		    ORDER BY position ASC LIMIT 1
		) orig ON ro.id IS NOT NULL
		LEFT JOIN LATERAL (
		    SELECT name FROM stops WHERE route_id = ro.id
		    ORDER BY position DESC LIMIT 1
		) dest ON ro.id IS NOT NULL
		WHERE res.code = $1
	`, code).Scan(
		&info.Code, &info.Status, &info.ExpiresAt, &info.CreatedAt,
		&info.PassengerName, &info.PassengerDocType, &info.PassengerDocNumber,
		&info.PassengerEmail, &info.PassengerPhone,
		&info.DocumentType, &info.PaymentMethod, &info.PaymentStatus,
		&billingSaleID, &amountCents,
		&info.RouteName, &tripInstanceID, &departureAt,
		&info.OriginStop, &info.DestStop,
	)
	if err != nil {
		return nil, err
	}
	info.BillingSaleID = billingSaleID
	info.TripInstanceID = tripInstanceID
	info.DepartureAt = departureAt
	if amountCents != nil {
		info.TotalAmount = float64(*amountCents) / 100.0
	}

	rows, err := r.Pool.Query(ctx, `
		SELECT ri.id,
		       vs.label,
		       COALESCE(t.code, ''),
		       COALESCE(t.status, '')
		FROM reservation_items ri
		JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		JOIN vehicle_seats vs ON vs.id = tsi.vehicle_seat_id
		LEFT JOIN tickets t ON t.trip_seat_inventory_id = tsi.id AND t.reservation_id = ri.reservation_id
		JOIN reservations res ON res.id = ri.reservation_id
		WHERE res.code = $1
		ORDER BY vs.label
	`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	info.Items = []domain.ReservationPublicItem{}
	for rows.Next() {
		var it domain.ReservationPublicItem
		if err := rows.Scan(&it.ID, &it.SeatLabel, &it.TicketCode, &it.TicketStatus); err != nil {
			return nil, err
		}
		info.Items = append(info.Items, it)
	}
	return &info, rows.Err()
}

// GetTripInstanceIDsByCode returns the distinct trip_instance_ids associated with a reservation code.
func (r *ReservationRepository) GetTripInstanceIDsByCode(ctx context.Context, code string) ([]int64, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT DISTINCT tsi.trip_instance_id
		FROM reservations res
		JOIN reservation_items ri ON ri.reservation_id = res.id
		JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		WHERE res.code = $1
	`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// VoidReservationResult lleva los identificadores afectados por la anulación
// para que el handler haga el broadcast WS y dispare la anulación remota en venta.
type VoidReservationResult struct {
	TripInstanceIDs []int64
	BillingSaleIDs  []int
}

// VoidReservation anula una venta confirmada: marca tickets como voided,
// libera los asientos del inventario, marca la reserva como cancelled y registra
// el motivo en el payment con el reviewer.
func (r *ReservationRepository) VoidReservation(ctx context.Context, code string, reviewerID int64, reason string) (*VoidReservationResult, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var reservationID int64
	var status string
	err = tx.QueryRow(ctx, `SELECT id, status FROM reservations WHERE code = $1`, code).Scan(&reservationID, &status)
	if err != nil {
		return nil, fmt.Errorf("reserva no encontrada: %w", err)
	}
	if status == "cancelled" || status == "expired" {
		return nil, fmt.Errorf("la reserva ya no está activa (estado: %s)", status)
	}

	// Recolectar trip_instance_ids afectados antes del cambio para el broadcast.
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT tsi.trip_instance_id
		  FROM reservation_items ri
		  JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		 WHERE ri.reservation_id = $1
	`, reservationID)
	if err != nil {
		return nil, err
	}
	var tripIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		tripIDs = append(tripIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 1. Marcar tickets como voided
	if _, err := tx.Exec(ctx, `
		UPDATE tickets SET status = 'voided', updated_at = now()
		 WHERE reservation_id = $1 AND status = 'issued'
	`, reservationID); err != nil {
		return nil, fmt.Errorf("error anulando tickets: %w", err)
	}

	// 2. Liberar asientos
	if _, err := tx.Exec(ctx, `
		UPDATE trip_seat_inventory
		   SET status = 'available', hold_expires_at = NULL, held_by = NULL, updated_at = now()
		 WHERE id IN (SELECT trip_seat_inventory_id FROM reservation_items WHERE reservation_id = $1)
	`, reservationID); err != nil {
		return nil, fmt.Errorf("error liberando asientos: %w", err)
	}

	// 2b. Eliminar reservation_items para permitir re-reserva del asiento
	if _, err := tx.Exec(ctx, `
		DELETE FROM reservation_items WHERE reservation_id = $1
	`, reservationID); err != nil {
		return nil, fmt.Errorf("error eliminando items de reserva: %w", err)
	}

	// 3. Marcar reserva como cancelled
	if _, err := tx.Exec(ctx, `
		UPDATE reservations SET status = 'cancelled', updated_at = now() WHERE id = $1
	`, reservationID); err != nil {
		return nil, fmt.Errorf("error actualizando reserva: %w", err)
	}

	// 4. Recoger billing_sale_ids antes de anular en pagos (para llamar a venta).
	saleRows, err := tx.Query(ctx, `
		SELECT DISTINCT billing_sale_id FROM payments
		 WHERE reservation_id = $1 AND billing_sale_id IS NOT NULL AND billing_sale_id > 0
	`, reservationID)
	if err != nil {
		return nil, err
	}
	var saleIDs []int
	for saleRows.Next() {
		var id int
		if err := saleRows.Scan(&id); err != nil {
			saleRows.Close()
			return nil, err
		}
		saleIDs = append(saleIDs, id)
	}
	saleRows.Close()
	if err := saleRows.Err(); err != nil {
		return nil, err
	}

	// 5. Marcar payments como voided y registrar motivo
	if _, err := tx.Exec(ctx, `
		UPDATE payments
		   SET status = 'voided',
		       rejection_reason = $3,
		       reviewed_by = COALESCE($2, reviewed_by),
		       reviewed_at = now(),
		       updated_at = now()
		 WHERE reservation_id = $1 AND status NOT IN ('rejected', 'voided', 'expired')
	`, reservationID, reviewerID, "ANULADA: "+reason); err != nil {
		return nil, fmt.Errorf("error actualizando pagos: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &VoidReservationResult{TripInstanceIDs: tripIDs, BillingSaleIDs: saleIDs}, nil
}

// UpdateReservationStatus updates the status of a reservation by code.
func (r *ReservationRepository) UpdateReservationStatus(ctx context.Context, code string, status string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE reservations SET status = $1, updated_at = now() WHERE code = $2`, status, code)
	return err
}

// CreateYapePayment creates a payment record for Yape direct with pending_verification status.
func (r *ReservationRepository) CreateYapePayment(ctx context.Context, p *domain.Payment) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO payments (
			reservation_id, amount_cents, currency, method, status, voucher_path,
			passenger_name, passenger_doc_type, passenger_doc_number, passenger_email,
			passenger_phone, passenger_address, document_type, route_name, seat_labels,
			trip_instance_id, price_per_seat
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`,
		p.ReservationID, p.AmountCents, p.Currency, p.Method, p.Status, p.VoucherPath,
		p.PassengerName, p.PassengerDocType, p.PassengerDocNumber, p.PassengerEmail,
		p.PassengerPhone, p.PassengerAddress, p.DocumentType, p.RouteName, p.SeatLabels,
		p.TripInstanceID, p.PricePerSeat,
	).Scan(&id)
	return id, err
}

// GetPendingVouchers returns all payments with pending_verification status for admin review.
func (r *ReservationRepository) GetPendingVouchers(ctx context.Context) ([]domain.VoucherReview, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT p.id, res.code, res.id, p.amount_cents,
			COALESCE(p.passenger_name,''), COALESCE(p.passenger_doc_type,''),
			COALESCE(p.passenger_doc_number,''), COALESCE(p.passenger_email,''),
			COALESCE(p.passenger_phone,''), COALESCE(p.document_type,'boleta'),
			COALESCE(p.route_name,''), p.seat_labels,
			p.trip_instance_id, COALESCE(p.price_per_seat,0),
			COALESCE(p.passenger_address,''),
			p.created_at, res.expires_at
		FROM payments p
		JOIN reservations res ON res.id = p.reservation_id
		WHERE p.status = 'pending_verification'
		ORDER BY p.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.VoucherReview
	for rows.Next() {
		var v domain.VoucherReview
		if err := rows.Scan(
			&v.PaymentID, &v.ReservationCode, &v.ReservationID, &v.AmountCents,
			&v.PassengerName, &v.PassengerDocType, &v.PassengerDocNumber, &v.PassengerEmail,
			&v.PassengerPhone, &v.DocumentType, &v.RouteName, &v.SeatLabels,
			&v.TripInstanceID, &v.PricePerSeat, &v.PassengerAddress,
			&v.CreatedAt, &v.HoldExpiresAt,
		); err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, rows.Err()
}

// GetPaymentByID returns a payment by its ID.
func (r *ReservationRepository) GetPaymentByID(ctx context.Context, id int64) (*domain.Payment, error) {
	var p domain.Payment
	err := r.Pool.QueryRow(ctx, `
		SELECT id, ticket_id, reservation_id, amount_cents, currency, method, reference, status,
			voucher_path, reviewed_by, reviewed_at, rejection_reason,
			COALESCE(passenger_name,''), COALESCE(passenger_doc_type,''),
			COALESCE(passenger_doc_number,''), COALESCE(passenger_email,''),
			COALESCE(passenger_phone,''), COALESCE(passenger_address,''),
			COALESCE(document_type,''), COALESCE(route_name,''), seat_labels,
			trip_instance_id, COALESCE(price_per_seat,0),
			billing_sale_id, billing_error, COALESCE(billing_attempts, 0),
			comprobante_email_sent_at,
			created_at, updated_at
		FROM payments WHERE id = $1
	`, id).Scan(
		&p.ID, &p.TicketID, &p.ReservationID, &p.AmountCents, &p.Currency, &p.Method, &p.Reference, &p.Status,
		&p.VoucherPath, &p.ReviewedBy, &p.ReviewedAt, &p.RejectionReason,
		&p.PassengerName, &p.PassengerDocType, &p.PassengerDocNumber, &p.PassengerEmail,
		&p.PassengerPhone, &p.PassengerAddress, &p.DocumentType, &p.RouteName, &p.SeatLabels,
		&p.TripInstanceID, &p.PricePerSeat,
		&p.BillingSaleID, &p.BillingError, &p.BillingAttempts,
		&p.ComprobanteEmailSentAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetPaymentBillingResult persists the outcome of a billing sale attempt on a payment.
// saleID nil + errMsg "" means "still in progress / not attempted yet"; the caller normally
// passes one or the other. attempts is the new total attempts counter.
func (r *ReservationRepository) SetPaymentBillingResult(ctx context.Context, paymentID int64, saleID *int, errMsg string, attempts int) error {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	_, err := r.Pool.Exec(ctx, `
		UPDATE payments
		   SET billing_sale_id  = COALESCE($2, billing_sale_id),
		       billing_error    = $3,
		       billing_attempts = $4,
		       updated_at       = now()
		 WHERE id = $1
	`, paymentID, saleID, errPtr, attempts)
	return err
}

// SeatHolder describes who currently occupies a seat (held/reserved/sold). Returned
// only to admin endpoints — public callers must not see passenger PII.
type SeatHolder struct {
	SeatLabel         string  `json:"seat_label"`
	SeatStatus        string  `json:"seat_status"`
	ReservationCode   string  `json:"reservation_code"`
	ReservationStatus string  `json:"reservation_status"`
	TicketCode        string  `json:"ticket_code,omitempty"`
	TicketStatus      string  `json:"ticket_status,omitempty"`
	PassengerName     string  `json:"passenger_name"`
	PassengerDocType  string  `json:"passenger_doc_type"`
	PassengerDocNum   string  `json:"passenger_doc_number"`
	PassengerEmail    string  `json:"passenger_email"`
	PassengerPhone    string  `json:"passenger_phone"`
	PaymentMethod     string  `json:"payment_method"`
	PaymentStatus     string  `json:"payment_status"`
	BillingSaleID     *int    `json:"billing_sale_id,omitempty"`
	CreatedAt         *string `json:"created_at,omitempty"`
}

// GetSeatHolder returns the passenger occupying a specific seat on a trip. It joins
// reservation_items → reservations → payments and falls back to tickets so it works
// for held, reserved and sold seats. Returns sql.ErrNoRows-equivalent (pgx.ErrNoRows)
// if the seat is not occupied or doesn't belong to the trip.
func (r *ReservationRepository) GetSeatHolder(ctx context.Context, tripID, seatInventoryID int64) (*SeatHolder, error) {
	var h SeatHolder
	var billingSaleID *int
	var ticketCode, ticketStatus, createdAt *string
	err := r.Pool.QueryRow(ctx, `
		SELECT vs.label,
		       tsi.status,
		       COALESCE(res.code, ''),
		       COALESCE(res.status, ''),
		       t.code,
		       t.status,
		       COALESCE(NULLIF(p.passenger_name,''), NULLIF(res.contact_name,''), NULLIF(res.contact_doc_number,''), ''),
		       COALESCE(NULLIF(p.passenger_doc_type,''), 'DNI', ''),
		       COALESCE(NULLIF(p.passenger_doc_number,''), NULLIF(res.contact_doc_number,''), ''),
		       COALESCE(p.passenger_email, ''),
		       COALESCE(NULLIF(p.passenger_phone,''), NULLIF(res.contact_phone,''), ''),
		       COALESCE(p.method, ''),
		       COALESCE(p.status, ''),
		       p.billing_sale_id,
		       to_char(res.created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM trip_seat_inventory tsi
		JOIN vehicle_seats vs ON vs.id = tsi.vehicle_seat_id
		LEFT JOIN reservation_items ri ON ri.trip_seat_inventory_id = tsi.id
		LEFT JOIN reservations res ON res.id = ri.reservation_id
		LEFT JOIN LATERAL (
		    SELECT * FROM payments WHERE reservation_id = res.id
		     ORDER BY created_at DESC LIMIT 1
		) p ON TRUE
		LEFT JOIN tickets t ON t.trip_seat_inventory_id = tsi.id AND t.status = 'issued'
		WHERE tsi.id = $1 AND tsi.trip_instance_id = $2
		ORDER BY res.created_at DESC NULLS LAST
		LIMIT 1
	`, seatInventoryID, tripID).Scan(
		&h.SeatLabel,
		&h.SeatStatus,
		&h.ReservationCode,
		&h.ReservationStatus,
		&ticketCode,
		&ticketStatus,
		&h.PassengerName,
		&h.PassengerDocType,
		&h.PassengerDocNum,
		&h.PassengerEmail,
		&h.PassengerPhone,
		&h.PaymentMethod,
		&h.PaymentStatus,
		&billingSaleID,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}
	if ticketCode != nil {
		h.TicketCode = *ticketCode
	}
	if ticketStatus != nil {
		h.TicketStatus = *ticketStatus
	}
	h.BillingSaleID = billingSaleID
	h.CreatedAt = createdAt
	return &h, nil
}

// MarkComprobanteEmailSent records that the comprobante email was delivered.
func (r *ReservationRepository) MarkComprobanteEmailSent(ctx context.Context, paymentID int64) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE payments SET comprobante_email_sent_at = now(), updated_at = now() WHERE id = $1
	`, paymentID)
	return err
}

// ListPaymentsPendingComprobante returns approved/completed payments whose comprobante is
// still missing (billing_sale_id NULL) and that haven't exhausted retries yet.
func (r *ReservationRepository) ListPaymentsPendingComprobante(ctx context.Context, maxAttempts int, limit int) ([]int64, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id FROM payments
		 WHERE status IN ('approved','completed')
		   AND document_type IS NOT NULL AND document_type <> ''
		   AND billing_sale_id IS NULL
		   AND billing_attempts < $1
		   AND passenger_email <> ''
		 ORDER BY created_at ASC
		 LIMIT $2
	`, maxAttempts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetPaymentByReservationCode returns the payment for a given reservation code.
func (r *ReservationRepository) GetPaymentByReservationCode(ctx context.Context, code string) (*domain.Payment, error) {
	var paymentID int64
	err := r.Pool.QueryRow(ctx, `
		SELECT p.id FROM payments p
		JOIN reservations res ON res.id = p.reservation_id
		WHERE res.code = $1
		ORDER BY p.created_at DESC LIMIT 1
	`, code).Scan(&paymentID)
	if err != nil {
		return nil, err
	}
	return r.GetPaymentByID(ctx, paymentID)
}

// SetReservationBillingSaleIDByCode persists a billing sale id across the
// payments of a reservation so void flows can cascade to tenant reliably.
func (r *ReservationRepository) SetReservationBillingSaleIDByCode(ctx context.Context, code string, saleID int) error {
	if code == "" || saleID <= 0 {
		return nil
	}
	_, err := r.Pool.Exec(ctx, `
		UPDATE payments p
		   SET billing_sale_id = COALESCE(p.billing_sale_id, $2),
		       billing_error = NULL,
		       updated_at = now()
		  FROM reservations r
		 WHERE r.id = p.reservation_id
		   AND r.code = $1
		   AND p.status IN ('approved', 'completed')
	`, code, saleID)
	return err
}

// ApprovePayment sets payment to approved. Returns false if already processed.
func (r *ReservationRepository) ApprovePayment(ctx context.Context, paymentID int64, reviewerID int64) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE payments
		SET status = 'approved', reviewed_by = $2, reviewed_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'pending_verification'
	`, paymentID, reviewerID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// RejectPayment rejects a payment and releases the held seats.
func (r *ReservationRepository) RejectPayment(ctx context.Context, paymentID int64, reviewerID int64, reason string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update payment
	tag, err := tx.Exec(ctx, `
		UPDATE payments
		SET status = 'rejected', reviewed_by = $2, reviewed_at = now(), rejection_reason = $3, updated_at = now()
		WHERE id = $1 AND status = 'pending_verification'
	`, paymentID, reviewerID, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pago no encontrado o ya procesado")
	}

	// Get reservation_id from payment
	var reservationID int64
	err = tx.QueryRow(ctx, `SELECT reservation_id FROM payments WHERE id = $1`, paymentID).Scan(&reservationID)
	if err != nil {
		return err
	}

	// Release held seats
	_, err = tx.Exec(ctx, `
		UPDATE trip_seat_inventory
		SET status = 'available', hold_expires_at = NULL, held_by = NULL, updated_at = now()
		WHERE id IN (SELECT trip_seat_inventory_id FROM reservation_items WHERE reservation_id = $1)
		AND status = 'held'
	`, reservationID)
	if err != nil {
		return err
	}

	// Cancel reservation
	_, err = tx.Exec(ctx, `UPDATE reservations SET status = 'cancelled', updated_at = now() WHERE id = $1`, reservationID)
	if err != nil {
		return err
	}

	// Delete reservation items to allow future re-reservation
	_, err = tx.Exec(ctx, `DELETE FROM reservation_items WHERE reservation_id = $1`, reservationID)
	if err != nil {
		return fmt.Errorf("error eliminando items de reserva: %w", err)
	}

	return tx.Commit(ctx)
}

// GetTripInstanceIDsForPayment returns the trip IDs affected by a payment's reservation.
func (r *ReservationRepository) GetTripInstanceIDsForPayment(ctx context.Context, paymentID int64) ([]int64, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT DISTINCT tsi.trip_instance_id
		FROM payments p
		JOIN reservation_items ri ON ri.reservation_id = p.reservation_id
		JOIN trip_seat_inventory tsi ON tsi.id = ri.trip_seat_inventory_id
		WHERE p.id = $1
	`, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// generateTicketCode genera un código TKT-XXXXXXXX.
func generateTicketCode() string {
	return "TKT-" + randomAlphanumeric(8)
}
