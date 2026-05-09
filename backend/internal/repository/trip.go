package repository

import (
	"context"
	"fmt"
	"time"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TripRepository acceso a trip_instances y trip_seat_inventory.
type TripRepository struct {
	Pool *pgxpool.Pool
}

// ListByRoute devuelve las instancias de viaje de una ruta desde fromDate.
func (r *TripRepository) ListByRoute(ctx context.Context, routeID int64, fromDate time.Time) ([]domain.TripInstance, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT ti.id, ti.trip_template_id, tt.route_id, r.name,
		       COALESCE(r.price_per_seat, 0), ti.departure_at, ti.status
		FROM trip_instances ti
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes r ON r.id = tt.route_id
		WHERE tt.route_id = $1 AND ti.departure_at >= $2 AND ti.status = 'scheduled'
		ORDER BY ti.departure_at
	`, routeID, fromDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.TripInstance
	for rows.Next() {
		var x domain.TripInstance
		if err := rows.Scan(&x.ID, &x.TripTemplateID, &x.RouteID, &x.RouteName, &x.PricePerSeat, &x.DepartureAt, &x.Status); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// GetByID devuelve una instancia por ID (para validar existencia).
func (r *TripRepository) GetByID(ctx context.Context, id int64) (*domain.TripInstance, error) {
	var x domain.TripInstance
	err := r.Pool.QueryRow(ctx, `
		SELECT ti.id, ti.trip_template_id, tt.route_id, r.name, COALESCE(r.price_per_seat, 0), ti.departure_at, ti.status
		FROM trip_instances ti
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes r ON r.id = tt.route_id
		WHERE ti.id = $1
	`, id).Scan(&x.ID, &x.TripTemplateID, &x.RouteID, &x.RouteName, &x.PricePerSeat, &x.DepartureAt, &x.Status)
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// ListSeatsByTripID devuelve el inventario de asientos de un viaje.
func (r *TripRepository) ListSeatsByTripID(ctx context.Context, tripID int64) ([]domain.SeatInfo, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT tsi.id, tsi.vehicle_seat_id, vs.label, vs.position,
		       vs.floor, vs.row_num, vs.col_num, vs.seat_type,
		       tsi.status, tsi.hold_expires_at
		FROM trip_seat_inventory tsi
		JOIN vehicle_seats vs ON vs.id = tsi.vehicle_seat_id
		WHERE tsi.trip_instance_id = $1
		ORDER BY vs.floor, vs.row_num, vs.col_num
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SeatInfo
	for rows.Next() {
		var x domain.SeatInfo
		if err := rows.Scan(&x.ID, &x.VehicleSeatID, &x.Label, &x.Position,
			&x.Floor, &x.RowNum, &x.ColNum, &x.SeatType,
			&x.Status, &x.HoldExpiresAt); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// GetTripSeatResponse devuelve los asientos con metadata del vehículo para el diagrama visual.
func (r *TripRepository) GetTripSeatResponse(ctx context.Context, tripID int64) (*domain.TripSeatResponse, error) {
	var resp domain.TripSeatResponse
	var vehicleID int64
	err := r.Pool.QueryRow(ctx, `
		SELECT ti.vehicle_id, v.floors, v.layout_cols, v.seat_type, COALESCE(v.name, ''),
		       COALESCE(ro.name, ''), COALESCE(ro.price_per_seat, 0)
		FROM trip_instances ti
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes ro ON ro.id = tt.route_id
		JOIN vehicles v ON v.id = ti.vehicle_id
		WHERE ti.id = $1
	`, tripID).Scan(&vehicleID, &resp.VehicleFloors, &resp.VehicleLayoutCols, &resp.VehicleSeatType, &resp.VehicleName, &resp.RouteName, &resp.PricePerSeat)
	if err != nil {
		return nil, err
	}

	seats, err := r.ListSeatsByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	resp.Seats = seats

	// Cargar elementos decorativos del vehículo (puede no haber).
	elemRows, err := r.Pool.Query(ctx, `
		SELECT id, vehicle_id, floor, row_num, col_num, kind, text
		FROM vehicle_layout_elements WHERE vehicle_id = $1
		ORDER BY floor, row_num, col_num
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer elemRows.Close()
	resp.LayoutElements = []domain.VehicleLayoutElement{}
	for elemRows.Next() {
		var e domain.VehicleLayoutElement
		if err := elemRows.Scan(&e.ID, &e.VehicleID, &e.Floor, &e.RowNum, &e.ColNum, &e.Kind, &e.Text); err != nil {
			return nil, err
		}
		resp.LayoutElements = append(resp.LayoutElements, e)
	}
	return &resp, nil
}

// ListAllInstances devuelve todas las instancias de viaje con joins. Soporta filtro from.
func (r *TripRepository) ListAllInstances(ctx context.Context, from time.Time) ([]domain.TripInstanceAdmin, error) {
	baseQuery := `
		SELECT ti.id, ti.trip_template_id, tt.route_id, ro.name, tt.vehicle_id, v.name,
		       ti.departure_at, ti.status, ti.created_at, ti.updated_at
		FROM trip_instances ti
		JOIN trip_templates tt ON tt.id = ti.trip_template_id
		JOIN routes ro ON ro.id = tt.route_id
		JOIN vehicles v ON v.id = tt.vehicle_id
	`
	var args []any
	if !from.IsZero() {
		baseQuery += " WHERE ti.departure_at >= $1"
		args = append(args, from)
	}
	baseQuery += " ORDER BY ti.departure_at"

	rows, err := r.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.TripInstanceAdmin
	for rows.Next() {
		var x domain.TripInstanceAdmin
		if err := rows.Scan(&x.ID, &x.TripTemplateID, &x.RouteID, &x.RouteName, &x.VehicleID, &x.VehicleName,
			&x.DepartureAt, &x.Status, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// CreateInstance crea una instancia de viaje desde una plantilla y genera el inventario de asientos.
func (r *TripRepository) CreateInstance(ctx context.Context, tripTemplateID int64, departureAt time.Time) (int64, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Obtener vehicle_id de la plantilla
	var vehicleID int64
	err = tx.QueryRow(ctx, `SELECT vehicle_id FROM trip_templates WHERE id = $1`, tripTemplateID).Scan(&vehicleID)
	if err != nil {
		return 0, fmt.Errorf("trip template not found: %w", err)
	}

	// Crear instancia
	var instanceID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO trip_instances (trip_template_id, vehicle_id, departure_at, status)
		VALUES ($1, $2, $3, 'scheduled') RETURNING id
	`, tripTemplateID, vehicleID, departureAt).Scan(&instanceID)
	if err != nil {
		return 0, fmt.Errorf("insert trip instance: %w", err)
	}

	// Generar inventario de asientos
	_, err = tx.Exec(ctx, `
		INSERT INTO trip_seat_inventory (trip_instance_id, vehicle_seat_id, status)
		SELECT $1, vs.id, 'available'
		FROM vehicle_seats vs WHERE vs.vehicle_id = $2
	`, instanceID, vehicleID)
	if err != nil {
		return 0, fmt.Errorf("insert seat inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return instanceID, nil
}

// UpdateInstanceStatus actualiza el estado de una instancia.
func (r *TripRepository) UpdateInstanceStatus(ctx context.Context, id int64, status string) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE trip_instances SET status = $2, updated_at = now() WHERE id = $1
	`, id, status)
	return err
}

// UpdateInstanceDeparture actualiza la hora de salida de una instancia.
func (r *TripRepository) UpdateInstanceDeparture(ctx context.Context, id int64, departureAt time.Time) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE trip_instances SET departure_at = $2, updated_at = now() WHERE id = $1
	`, id, departureAt)
	return err
}
