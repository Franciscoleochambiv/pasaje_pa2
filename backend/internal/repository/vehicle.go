package repository

import (
	"context"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VehicleRepository acceso a datos de vehículos y sus asientos.
type VehicleRepository struct {
	Pool *pgxpool.Pool
}

// List devuelve todos los vehículos.
func (r *VehicleRepository) List(ctx context.Context) ([]domain.Vehicle, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, plate, name, capacity, floors, layout_cols, seat_type, layout_id, active, created_at, updated_at
		FROM vehicles ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Vehicle
	for rows.Next() {
		var x domain.Vehicle
		if err := rows.Scan(&x.ID, &x.Plate, &x.Name, &x.Capacity, &x.Floors, &x.LayoutCols, &x.SeatType, &x.LayoutID, &x.Active, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// Get devuelve un vehículo por su ID.
func (r *VehicleRepository) Get(ctx context.Context, id int64) (*domain.Vehicle, error) {
	var v domain.Vehicle
	err := r.Pool.QueryRow(ctx, `
		SELECT id, plate, name, capacity, floors, layout_cols, seat_type, layout_id, active, created_at, updated_at
		FROM vehicles WHERE id = $1
	`, id).Scan(&v.ID, &v.Plate, &v.Name, &v.Capacity, &v.Floors, &v.LayoutCols, &v.SeatType, &v.LayoutID, &v.Active, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Create inserta un vehículo y devuelve su ID.
func (r *VehicleRepository) Create(ctx context.Context, v *domain.Vehicle) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO vehicles (plate, name, capacity, floors, layout_cols, seat_type)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, v.Plate, v.Name, v.Capacity, v.Floors, v.LayoutCols, v.SeatType).Scan(&id)
	return id, err
}

// Update actualiza un vehículo. Los campos que lleguen vacíos/cero se preservan
// con su valor actual para no corromper la geometría del bus (floors, layout_cols,
// seat_type) cuando el handler envía un payload parcial.
func (r *VehicleRepository) Update(ctx context.Context, v *domain.Vehicle) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE vehicles SET
			plate       = CASE WHEN $2 = ''  THEN plate       ELSE $2 END,
			name        = COALESCE($3, name),
			capacity    = CASE WHEN $4 = 0   THEN capacity    ELSE $4 END,
			floors      = CASE WHEN $6 = 0   THEN floors      ELSE $6 END,
			layout_cols = CASE WHEN $7 = 0   THEN layout_cols ELSE $7 END,
			seat_type   = CASE WHEN $8 = ''  THEN seat_type   ELSE $8 END,
			active      = $5,
			updated_at  = now()
		WHERE id = $1
	`, v.ID, v.Plate, v.Name, v.Capacity, v.Active, v.Floors, v.LayoutCols, v.SeatType)
	return err
}

// Deactivate marca un vehículo como inactivo sin tocar el resto de columnas.
func (r *VehicleRepository) Deactivate(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `UPDATE vehicles SET active = false, updated_at = now() WHERE id = $1`, id)
	return err
}

// CreateSeat inserta un asiento de vehículo.
func (r *VehicleRepository) CreateSeat(ctx context.Context, s *domain.VehicleSeat) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO vehicle_seats (vehicle_id, label, position, floor, row_num, col_num, seat_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, s.VehicleID, s.Label, s.Position, s.Floor, s.RowNum, s.ColNum, s.SeatType).Scan(&id)
	return id, err
}

// ListSeats devuelve los asientos de un vehículo.
func (r *VehicleRepository) ListSeats(ctx context.Context, vehicleID int64) ([]domain.VehicleSeat, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, vehicle_id, label, position, floor, row_num, col_num, seat_type
		FROM vehicle_seats
		WHERE vehicle_id = $1 ORDER BY floor, row_num, col_num
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.VehicleSeat
	for rows.Next() {
		var x domain.VehicleSeat
		if err := rows.Scan(&x.ID, &x.VehicleID, &x.Label, &x.Position, &x.Floor, &x.RowNum, &x.ColNum, &x.SeatType); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}
