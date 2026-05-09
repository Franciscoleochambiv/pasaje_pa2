package repository

import (
	"context"
	"fmt"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ParcelRepository acceso a datos de encomiendas.
type ParcelRepository struct {
	Pool *pgxpool.Pool
}

// CancelParcelResult expone los datos necesarios después de anular.
type CancelParcelResult struct {
	Code          string
	BillingSaleID *int64
}

// Create inserta una encomienda y su primer tracking.
func (r *ParcelRepository) Create(ctx context.Context, p *domain.Parcel) (int64, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO parcels (
			code, trip_instance_id, origin_stop_id, dest_stop_id,
			sender_name, sender_doc_type, sender_doc_number, sender_phone,
			receiver_name, receiver_doc_type, receiver_doc_number, receiver_phone,
			package_count, weight_kg, description,
			amount_cents, currency, payment_mode, payment_status, payment_method,
			billing_doc_type, billing_email, billing_ruc, billing_razon_social, billing_address,
			billing_sale_id, status, registered_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28
		) RETURNING id
	`,
		p.Code, p.TripInstanceID, p.OriginStopID, p.DestStopID,
		p.SenderName, p.SenderDocType, p.SenderDocNumber, p.SenderPhone,
		p.ReceiverName, p.ReceiverDocType, p.ReceiverDocNumber, p.ReceiverPhone,
		p.PackageCount, p.WeightKg, p.Description,
		p.AmountCents, p.Currency, p.PaymentMode, p.PaymentStatus, p.PaymentMethod,
		p.BillingDocType, p.BillingEmail, p.BillingRuc, p.BillingRazonSocial, p.BillingAddress,
		p.BillingSaleID, p.Status, p.RegisteredBy,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert parcel: %w", err)
	}

	// Insertar tracking inicial
	_, err = tx.Exec(ctx, `
		INSERT INTO parcel_tracking (parcel_id, status, location, notes, user_id)
		VALUES ($1, 'registered', $2, 'Encomienda registrada', $3)
	`, id, "", p.RegisteredBy)
	if err != nil {
		return 0, fmt.Errorf("insert initial tracking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

// Update actualiza los datos editables de una encomienda.
func (r *ParcelRepository) Update(ctx context.Context, p *domain.Parcel) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE parcels SET
			sender_name=$2, sender_doc_type=$3, sender_doc_number=$4, sender_phone=$5,
			receiver_name=$6, receiver_doc_type=$7, receiver_doc_number=$8, receiver_phone=$9,
			package_count=$10, weight_kg=$11, description=$12,
			amount_cents=$13, payment_mode=$14,
			billing_doc_type=$15, billing_email=$16, billing_ruc=$17, billing_razon_social=$18, billing_address=$19,
			updated_at=now()
		WHERE id=$1
	`,
		p.ID,
		p.SenderName, p.SenderDocType, p.SenderDocNumber, p.SenderPhone,
		p.ReceiverName, p.ReceiverDocType, p.ReceiverDocNumber, p.ReceiverPhone,
		p.PackageCount, p.WeightKg, p.Description,
		p.AmountCents, p.PaymentMode,
		p.BillingDocType, p.BillingEmail, p.BillingRuc, p.BillingRazonSocial, p.BillingAddress,
	)
	return err
}

// GetByID devuelve una encomienda por ID con nombres de paradas.
func (r *ParcelRepository) GetByID(ctx context.Context, id int64) (*domain.Parcel, error) {
	var p domain.Parcel
	var billingSaleID *int64
	err := r.Pool.QueryRow(ctx, `
		SELECT p.id, p.code, p.trip_instance_id, p.origin_stop_id, p.dest_stop_id,
			os.name, ds.name,
			p.sender_name, p.sender_doc_type, p.sender_doc_number, COALESCE(p.sender_phone,''),
			p.receiver_name, p.receiver_doc_type, p.receiver_doc_number, p.receiver_phone,
			p.package_count, COALESCE(p.weight_kg,0), COALESCE(p.description,''),
			p.amount_cents, p.currency, p.payment_mode, p.payment_status, COALESCE(p.payment_method,''),
			p.billing_doc_type, p.billing_email, COALESCE(p.billing_ruc,''), COALESCE(p.billing_razon_social,''), COALESCE(p.billing_address,''),
			p.billing_sale_id,
			p.status, p.registered_by,
			COALESCE(r.name,''), COALESCE(ti.departure_at::text,''),
			p.created_at, p.updated_at
		FROM parcels p
		JOIN stops os ON os.id = p.origin_stop_id
		JOIN stops ds ON ds.id = p.dest_stop_id
		JOIN trip_instances ti ON ti.id = p.trip_instance_id
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes r ON r.id = tt.route_id
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.Code, &p.TripInstanceID, &p.OriginStopID, &p.DestStopID,
		&p.OriginStopName, &p.DestStopName,
		&p.SenderName, &p.SenderDocType, &p.SenderDocNumber, &p.SenderPhone,
		&p.ReceiverName, &p.ReceiverDocType, &p.ReceiverDocNumber, &p.ReceiverPhone,
		&p.PackageCount, &p.WeightKg, &p.Description,
		&p.AmountCents, &p.Currency, &p.PaymentMode, &p.PaymentStatus, &p.PaymentMethod,
		&p.BillingDocType, &p.BillingEmail, &p.BillingRuc, &p.BillingRazonSocial, &p.BillingAddress,
		&billingSaleID,
		&p.Status, &p.RegisteredBy,
		&p.RouteName, &p.DepartureAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	p.BillingSaleID = billingSaleID
	return &p, err
}

// GetByCode devuelve una encomienda por código.
func (r *ParcelRepository) GetByCode(ctx context.Context, code string) (*domain.Parcel, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `SELECT id FROM parcels WHERE code = $1`, code).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// List devuelve encomiendas con filtros opcionales.
func (r *ParcelRepository) List(ctx context.Context, status string, tripInstanceID int64, dateFrom, dateTo string, limit, offset int) ([]domain.Parcel, int, error) {
	baseWhere := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		baseWhere += fmt.Sprintf(" AND p.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if tripInstanceID > 0 {
		baseWhere += fmt.Sprintf(" AND p.trip_instance_id = $%d", argIdx)
		args = append(args, tripInstanceID)
		argIdx++
	}
	if dateFrom != "" {
		baseWhere += fmt.Sprintf(" AND p.created_at >= $%d::date", argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		baseWhere += fmt.Sprintf(" AND p.created_at < ($%d::date + interval '1 day')", argIdx)
		args = append(args, dateTo)
		argIdx++
	}

	// Total count
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM parcels p %s`, baseWhere)
	err := r.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	query := fmt.Sprintf(`
		SELECT p.id, p.code, p.trip_instance_id, p.origin_stop_id, p.dest_stop_id,
			os.name, ds.name,
			p.sender_name, p.sender_doc_type, p.sender_doc_number, COALESCE(p.sender_phone,''),
			p.receiver_name, p.receiver_doc_type, p.receiver_doc_number, p.receiver_phone,
			p.package_count, COALESCE(p.weight_kg,0), COALESCE(p.description,''),
			p.amount_cents, p.currency, p.payment_mode, p.payment_status, COALESCE(p.payment_method,''),
			p.billing_doc_type, p.billing_email,
			p.billing_sale_id,
			p.status, p.registered_by,
			COALESCE(r.name,''), COALESCE(ti.departure_at::text,''),
			p.created_at, p.updated_at
		FROM parcels p
		JOIN stops os ON os.id = p.origin_stop_id
		JOIN stops ds ON ds.id = p.dest_stop_id
		JOIN trip_instances ti ON ti.id = p.trip_instance_id
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes r ON r.id = tt.route_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.Parcel
	for rows.Next() {
		var p domain.Parcel
		var billingSaleID *int64
		if err := rows.Scan(
			&p.ID, &p.Code, &p.TripInstanceID, &p.OriginStopID, &p.DestStopID,
			&p.OriginStopName, &p.DestStopName,
			&p.SenderName, &p.SenderDocType, &p.SenderDocNumber, &p.SenderPhone,
			&p.ReceiverName, &p.ReceiverDocType, &p.ReceiverDocNumber, &p.ReceiverPhone,
			&p.PackageCount, &p.WeightKg, &p.Description,
			&p.AmountCents, &p.Currency, &p.PaymentMode, &p.PaymentStatus, &p.PaymentMethod,
			&p.BillingDocType, &p.BillingEmail,
			&billingSaleID,
			&p.Status, &p.RegisteredBy,
			&p.RouteName, &p.DepartureAt,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		p.BillingSaleID = billingSaleID
		list = append(list, p)
	}
	return list, total, rows.Err()
}

// ListByTrip devuelve encomiendas de un viaje específico.
func (r *ParcelRepository) ListByTrip(ctx context.Context, tripInstanceID int64) ([]domain.Parcel, error) {
	list, _, err := r.List(ctx, "", tripInstanceID, "", "", 1000, 0)
	return list, err
}

// UpdateStatus actualiza el estado de una encomienda e inserta tracking con lock.
func (r *ParcelRepository) UpdateStatus(ctx context.Context, id int64, status, location, notes string, userID *int64) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock row to prevent concurrent modifications
	var currentStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM parcels WHERE id = $1 FOR UPDATE`, id).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("encomienda no encontrada: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE parcels SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO parcel_tracking (parcel_id, status, location, notes, user_id)
		VALUES ($1, $2, $3, $4, $5)
	`, id, status, location, notes, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// MarkPaid marca una encomienda como pagada con lock para evitar doble pago.
func (r *ParcelRepository) MarkPaid(ctx context.Context, id int64, paymentMethod string, billingSaleID *int64) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock and verify not already paid
	var paymentStatus string
	err = tx.QueryRow(ctx, `SELECT payment_status FROM parcels WHERE id = $1 FOR UPDATE`, id).Scan(&paymentStatus)
	if err != nil {
		return fmt.Errorf("encomienda no encontrada: %w", err)
	}
	if paymentStatus == "paid" {
		return fmt.Errorf("la encomienda ya fue pagada")
	}

	_, err = tx.Exec(ctx, `
		UPDATE parcels SET payment_status = 'paid', payment_method = $2, billing_sale_id = $3, updated_at = now()
		WHERE id = $1
	`, id, paymentMethod, billingSaleID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetBillingSaleID guarda el ID de venta del sistema de facturación (usado para pedidos).
func (r *ParcelRepository) SetBillingSaleID(ctx context.Context, id int64, saleID int64) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE parcels SET billing_sale_id = $2, updated_at = now() WHERE id = $1
	`, id, saleID)
	return err
}

// Cancel anula una encomienda e inserta tracking con validación.
func (r *ParcelRepository) Cancel(ctx context.Context, id int64, notes string, userID *int64) (*CancelParcelResult, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock and verify cancellable
	var currentStatus string
	var code string
	var billingSaleID *int64
	err = tx.QueryRow(ctx, `SELECT code, status, billing_sale_id FROM parcels WHERE id = $1 FOR UPDATE`, id).Scan(&code, &currentStatus, &billingSaleID)
	if err != nil {
		return nil, fmt.Errorf("encomienda no encontrada: %w", err)
	}
	if currentStatus == "delivered" {
		return nil, fmt.Errorf("no se puede anular una encomienda entregada")
	}
	if currentStatus == "cancelled" {
		return nil, fmt.Errorf("la encomienda ya está anulada")
	}

	_, err = tx.Exec(ctx, `UPDATE parcels SET status = 'cancelled', updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO parcel_tracking (parcel_id, status, location, notes, user_id)
		VALUES ($1, 'cancelled', '', $2, $3)
	`, id, notes, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &CancelParcelResult{
		Code:          code,
		BillingSaleID: billingSaleID,
	}, nil
}

// GetTracking devuelve el historial de movimientos de una encomienda.
func (r *ParcelRepository) GetTracking(ctx context.Context, parcelID int64) ([]domain.ParcelTracking, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT pt.id, pt.parcel_id, pt.status, COALESCE(pt.location,''), COALESCE(pt.notes,''),
			pt.user_id, COALESCE(u.name,''),
			pt.created_at
		FROM parcel_tracking pt
		LEFT JOIN users u ON u.id = pt.user_id
		WHERE pt.parcel_id = $1
		ORDER BY pt.created_at ASC
	`, parcelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ParcelTracking
	for rows.Next() {
		var t domain.ParcelTracking
		if err := rows.Scan(&t.ID, &t.ParcelID, &t.Status, &t.Location, &t.Notes, &t.UserID, &t.UserName, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// GetPublicInfo devuelve la información pública de tracking.
func (r *ParcelRepository) GetPublicInfo(ctx context.Context, code string) (*domain.ParcelPublicInfo, error) {
	p, err := r.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	tracking, err := r.GetTracking(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	// Limpiar datos internos del tracking para la vista pública
	publicTracking := make([]domain.ParcelTracking, len(tracking))
	for i, t := range tracking {
		publicTracking[i] = domain.ParcelTracking{
			ID:        t.ID,
			ParcelID:  t.ParcelID,
			Status:    t.Status,
			Location:  t.Location,
			Notes:     t.Notes,
			CreatedAt: t.CreatedAt,
			// UserID y UserName no se exponen en público
		}
	}

	return &domain.ParcelPublicInfo{
		Code:          p.Code,
		Status:        p.Status,
		OriginStop:    p.OriginStopName,
		DestStop:      p.DestStopName,
		ReceiverName:  p.ReceiverName,
		PackageCount:  p.PackageCount,
		WeightKg:      p.WeightKg,
		Description:   p.Description,
		PaymentMode:   p.PaymentMode,
		PaymentStatus: p.PaymentStatus,
		CreatedAt:     p.CreatedAt,
		Tracking:      publicTracking,
	}, nil
}

// GenerateParcelCode genera un código ENC-XXXXXXXX.
func GenerateParcelCode() string {
	return "ENC-" + randomAlphanumeric(8)
}
