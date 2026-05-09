package repository

import (
	"context"
	"fmt"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BusLayoutRepository acceso a datos de plantillas de bus.
type BusLayoutRepository struct {
	Pool *pgxpool.Pool
}

func (r *BusLayoutRepository) List(ctx context.Context) ([]domain.BusLayout, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, name, brand, model, description, floors, layout_cols, seat_type,
		       preview_image_url, active, created_at, updated_at
		FROM bus_layouts ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.BusLayout
	for rows.Next() {
		var x domain.BusLayout
		if err := rows.Scan(&x.ID, &x.Name, &x.Brand, &x.Model, &x.Description,
			&x.Floors, &x.LayoutCols, &x.SeatType, &x.PreviewImageURL,
			&x.Active, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

func (r *BusLayoutRepository) Get(ctx context.Context, id int64) (*domain.BusLayout, error) {
	var x domain.BusLayout
	err := r.Pool.QueryRow(ctx, `
		SELECT id, name, brand, model, description, floors, layout_cols, seat_type,
		       preview_image_url, active, created_at, updated_at
		FROM bus_layouts WHERE id = $1
	`, id).Scan(&x.ID, &x.Name, &x.Brand, &x.Model, &x.Description,
		&x.Floors, &x.LayoutCols, &x.SeatType, &x.PreviewImageURL,
		&x.Active, &x.CreatedAt, &x.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *BusLayoutRepository) Create(ctx context.Context, l *domain.BusLayout) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO bus_layouts (name, brand, model, description, floors, layout_cols, seat_type, preview_image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`, l.Name, l.Brand, l.Model, l.Description, l.Floors, l.LayoutCols, l.SeatType, l.PreviewImageURL).Scan(&id)
	return id, err
}

func (r *BusLayoutRepository) Update(ctx context.Context, l *domain.BusLayout) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE bus_layouts SET
			name = $2, brand = $3, model = $4, description = $5,
			floors = $6, layout_cols = $7, seat_type = $8, preview_image_url = $9,
			active = $10, updated_at = now()
		WHERE id = $1
	`, l.ID, l.Name, l.Brand, l.Model, l.Description,
		l.Floors, l.LayoutCols, l.SeatType, l.PreviewImageURL, l.Active)
	return err
}

func (r *BusLayoutRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM bus_layouts WHERE id = $1`, id)
	return err
}

func (r *BusLayoutRepository) ListSeats(ctx context.Context, layoutID int64) ([]domain.BusLayoutSeat, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, layout_id, label, position, floor, row_num, col_num, seat_type
		FROM bus_layout_seats WHERE layout_id = $1
		ORDER BY floor, row_num, col_num
	`, layoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.BusLayoutSeat
	for rows.Next() {
		var x domain.BusLayoutSeat
		if err := rows.Scan(&x.ID, &x.LayoutID, &x.Label, &x.Position,
			&x.Floor, &x.RowNum, &x.ColNum, &x.SeatType); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

func (r *BusLayoutRepository) ListElements(ctx context.Context, layoutID int64) ([]domain.BusLayoutElement, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, layout_id, floor, row_num, col_num, kind, text
		FROM bus_layout_elements WHERE layout_id = $1
		ORDER BY floor, row_num, col_num
	`, layoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.BusLayoutElement
	for rows.Next() {
		var x domain.BusLayoutElement
		if err := rows.Scan(&x.ID, &x.LayoutID, &x.Floor, &x.RowNum, &x.ColNum,
			&x.Kind, &x.Text); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// GetFull devuelve la plantilla con sus asientos y elementos.
func (r *BusLayoutRepository) GetFull(ctx context.Context, id int64) (*domain.BusLayoutFull, error) {
	layout, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	seats, err := r.ListSeats(ctx, id)
	if err != nil {
		return nil, err
	}
	if seats == nil {
		seats = []domain.BusLayoutSeat{}
	}
	elements, err := r.ListElements(ctx, id)
	if err != nil {
		return nil, err
	}
	if elements == nil {
		elements = []domain.BusLayoutElement{}
	}
	return &domain.BusLayoutFull{Layout: *layout, Seats: seats, Elements: elements}, nil
}

// SaveFull reemplaza atómicamente los asientos y elementos de la plantilla.
// El layout (metadata) se actualiza por separado vía Update.
func (r *BusLayoutRepository) SaveFull(ctx context.Context, layoutID int64, seats []domain.BusLayoutSeat, elements []domain.BusLayoutElement) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM bus_layout_seats WHERE layout_id = $1`, layoutID); err != nil {
		return fmt.Errorf("delete seats: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bus_layout_elements WHERE layout_id = $1`, layoutID); err != nil {
		return fmt.Errorf("delete elements: %w", err)
	}

	for _, s := range seats {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bus_layout_seats (layout_id, label, position, floor, row_num, col_num, seat_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, layoutID, s.Label, s.Position, s.Floor, s.RowNum, s.ColNum, s.SeatType); err != nil {
			return fmt.Errorf("insert seat %s: %w", s.Label, err)
		}
	}
	for _, e := range elements {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bus_layout_elements (layout_id, floor, row_num, col_num, kind, text)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, layoutID, e.Floor, e.RowNum, e.ColNum, e.Kind, e.Text); err != nil {
			return fmt.Errorf("insert element %s: %w", e.Kind, err)
		}
	}

	return tx.Commit(ctx)
}

// CloneToVehicle copia los asientos y elementos de la plantilla al vehículo dado.
// Se llama al crear un vehículo con layout_id, dentro de la misma transacción del Create.
func (r *BusLayoutRepository) CloneToVehicle(ctx context.Context, layoutID, vehicleID int64) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Limpiar lo que pueda existir (al re-aplicar plantilla a un vehículo).
	if _, err := tx.Exec(ctx, `DELETE FROM vehicle_seats WHERE vehicle_id = $1`, vehicleID); err != nil {
		return fmt.Errorf("clean vehicle_seats: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM vehicle_layout_elements WHERE vehicle_id = $1`, vehicleID); err != nil {
		return fmt.Errorf("clean vehicle_layout_elements: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO vehicle_seats (vehicle_id, label, position, floor, row_num, col_num, seat_type)
		SELECT $1, label, position, floor, row_num, col_num, seat_type
		FROM bus_layout_seats WHERE layout_id = $2
	`, vehicleID, layoutID); err != nil {
		return fmt.Errorf("clone seats: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO vehicle_layout_elements (vehicle_id, floor, row_num, col_num, kind, text)
		SELECT $1, floor, row_num, col_num, kind, text
		FROM bus_layout_elements WHERE layout_id = $2
	`, vehicleID, layoutID); err != nil {
		return fmt.Errorf("clone elements: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE vehicles SET layout_id = $1 WHERE id = $2`, layoutID, vehicleID); err != nil {
		return fmt.Errorf("set layout_id: %w", err)
	}

	return tx.Commit(ctx)
}

// CreateFromVehicle genera una plantilla nueva tomando el layout actual de un vehículo.
func (r *BusLayoutRepository) CreateFromVehicle(ctx context.Context, vehicleID int64, name, brand, model, description, previewImageURL string) (int64, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var floors, layoutCols int
	var seatType string
	err = tx.QueryRow(ctx, `SELECT floors, layout_cols, seat_type FROM vehicles WHERE id = $1`, vehicleID).
		Scan(&floors, &layoutCols, &seatType)
	if err != nil {
		return 0, fmt.Errorf("get vehicle: %w", err)
	}

	var layoutID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO bus_layouts (name, brand, model, description, floors, layout_cols, seat_type, preview_image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`, name, brand, model, description, floors, layoutCols, seatType, previewImageURL).Scan(&layoutID)
	if err != nil {
		return 0, fmt.Errorf("insert layout: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO bus_layout_seats (layout_id, label, position, floor, row_num, col_num, seat_type)
		SELECT $1, label, position, floor, row_num, col_num, seat_type
		FROM vehicle_seats WHERE vehicle_id = $2
	`, layoutID, vehicleID); err != nil {
		return 0, fmt.Errorf("clone seats: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO bus_layout_elements (layout_id, floor, row_num, col_num, kind, text)
		SELECT $1, floor, row_num, col_num, kind, text
		FROM vehicle_layout_elements WHERE vehicle_id = $2
	`, layoutID, vehicleID); err != nil {
		return 0, fmt.Errorf("clone elements: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return layoutID, nil
}

// ListVehicleElements devuelve los elementos decorativos de un vehículo.
func (r *BusLayoutRepository) ListVehicleElements(ctx context.Context, vehicleID int64) ([]domain.VehicleLayoutElement, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, vehicle_id, floor, row_num, col_num, kind, text
		FROM vehicle_layout_elements WHERE vehicle_id = $1
		ORDER BY floor, row_num, col_num
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.VehicleLayoutElement
	for rows.Next() {
		var x domain.VehicleLayoutElement
		if err := rows.Scan(&x.ID, &x.VehicleID, &x.Floor, &x.RowNum, &x.ColNum,
			&x.Kind, &x.Text); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}
