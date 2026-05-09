package repository

import (
	"context"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RouteRepository acceso a datos de rutas.
type RouteRepository struct {
	Pool *pgxpool.Pool
}

// List devuelve todas las rutas activas.
func (r *RouteRepository) List(ctx context.Context) ([]domain.Route, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, name, code, active, COALESCE(price_per_seat, 0) FROM routes WHERE active = true ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Route
	for rows.Next() {
		var x domain.Route
		if err := rows.Scan(&x.ID, &x.Name, &x.Code, &x.Active, &x.PricePerSeat); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// Create inserta una ruta y devuelve su ID.
func (r *RouteRepository) Create(ctx context.Context, route *domain.Route) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO routes (name, code, price_per_seat) VALUES ($1, $2, $3) RETURNING id
	`, route.Name, route.Code, route.PricePerSeat).Scan(&id)
	return id, err
}

// Update actualiza una ruta.
func (r *RouteRepository) Update(ctx context.Context, route *domain.Route) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE routes SET name = $2, code = $3, active = $4, price_per_seat = $5, updated_at = now() WHERE id = $1
	`, route.ID, route.Name, route.Code, route.Active, route.PricePerSeat)
	return err
}

// Deactivate desactiva una ruta (soft delete).
func (r *RouteRepository) Deactivate(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE routes SET active = false, updated_at = now() WHERE id = $1
	`, id)
	return err
}

// ── Stops ────────────────────────────────────────────────────────────────

// ListStops devuelve las paradas de una ruta ordenadas por posición.
func (r *RouteRepository) ListStops(ctx context.Context, routeID int64) ([]domain.Stop, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, route_id, name, code, position FROM stops WHERE route_id = $1 ORDER BY position
	`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Stop
	for rows.Next() {
		var s domain.Stop
		if err := rows.Scan(&s.ID, &s.RouteID, &s.Name, &s.Code, &s.Position); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// CreateStop inserta una parada y devuelve su ID.
func (r *RouteRepository) CreateStop(ctx context.Context, stop *domain.Stop) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO stops (route_id, name, code, position) VALUES ($1, $2, $3, $4) RETURNING id
	`, stop.RouteID, stop.Name, stop.Code, stop.Position).Scan(&id)
	return id, err
}

// UpdateStop actualiza una parada existente.
func (r *RouteRepository) UpdateStop(ctx context.Context, stop *domain.Stop) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE stops SET name = $2, code = $3, position = $4, updated_at = now() WHERE id = $1
	`, stop.ID, stop.Name, stop.Code, stop.Position)
	return err
}

// DeleteStop elimina una parada por ID.
func (r *RouteRepository) DeleteStop(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `
		DELETE FROM stops WHERE id = $1
	`, id)
	return err
}

// ── Route Segments ───────────────────────────────────────────────────────

// ListSegments devuelve los tramos de una ruta con nombres de paradas.
func (r *RouteRepository) ListSegments(ctx context.Context, routeID int64) ([]domain.RouteSegment, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT rs.id, rs.route_id, rs.origin_stop_id, rs.dest_stop_id, rs.price,
		       o.name, d.name
		  FROM route_segments rs
		  JOIN stops o ON o.id = rs.origin_stop_id
		  JOIN stops d ON d.id = rs.dest_stop_id
		 WHERE rs.route_id = $1
		 ORDER BY o.position, d.position
	`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.RouteSegment
	for rows.Next() {
		var s domain.RouteSegment
		if err := rows.Scan(&s.ID, &s.RouteID, &s.OriginStopID, &s.DestStopID, &s.Price,
			&s.OriginStopName, &s.DestStopName); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// UpsertSegment inserta o actualiza un tramo y devuelve su ID.
func (r *RouteRepository) UpsertSegment(ctx context.Context, seg *domain.RouteSegment) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO route_segments (route_id, origin_stop_id, dest_stop_id, price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (route_id, origin_stop_id, dest_stop_id)
		DO UPDATE SET price = EXCLUDED.price, updated_at = now()
		RETURNING id
	`, seg.RouteID, seg.OriginStopID, seg.DestStopID, seg.Price).Scan(&id)
	return id, err
}

// DeleteSegment elimina un tramo por ID.
func (r *RouteRepository) DeleteSegment(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `
		DELETE FROM route_segments WHERE id = $1
	`, id)
	return err
}

// GetSegmentPrice devuelve el precio de un tramo específico.
func (r *RouteRepository) GetSegmentPrice(ctx context.Context, routeID, originStopID, destStopID int64) (float64, error) {
	var price float64
	err := r.Pool.QueryRow(ctx, `
		SELECT price FROM route_segments
		 WHERE route_id = $1 AND origin_stop_id = $2 AND dest_stop_id = $3
	`, routeID, originStopID, destStopID).Scan(&price)
	return price, err
}
