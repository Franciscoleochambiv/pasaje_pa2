package repository

import (
	"context"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TripTemplateRepository acceso a datos de plantillas de viaje.
type TripTemplateRepository struct {
	Pool *pgxpool.Pool
}

// List devuelve todas las plantillas de viaje con datos de ruta y vehículo.
// departure_time es nullable desde la migration 17; lo serializamos como string
// vacío cuando viene NULL para que el front pueda renderizarlo sin chequeos.
func (r *TripTemplateRepository) List(ctx context.Context) ([]domain.TripTemplate, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT tt.id, tt.route_id, tt.vehicle_id, tt.name,
		       COALESCE(tt.departure_time::text, '') AS departure_time,
		       tt.active, ro.name, v.name, tt.created_at, tt.updated_at
		FROM trip_templates tt
		JOIN routes ro ON ro.id = tt.route_id
		JOIN vehicles v ON v.id = tt.vehicle_id
		ORDER BY tt.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.TripTemplate
	for rows.Next() {
		var x domain.TripTemplate
		if err := rows.Scan(&x.ID, &x.RouteID, &x.VehicleID, &x.Name, &x.DepartureTime,
			&x.Active, &x.RouteName, &x.VehicleName, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

// Create inserta una plantilla de viaje y devuelve su ID.
// Si DepartureTime es "" se guarda NULL (la hora va en cada trip_instance).
func (r *TripTemplateRepository) Create(ctx context.Context, t *domain.TripTemplate) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO trip_templates (route_id, vehicle_id, name, departure_time)
		VALUES ($1, $2, $3, NULLIF($4, '')::time) RETURNING id
	`, t.RouteID, t.VehicleID, t.Name, t.DepartureTime).Scan(&id)
	return id, err
}

// Update actualiza una plantilla de viaje. Acepta DepartureTime vacío → NULL.
func (r *TripTemplateRepository) Update(ctx context.Context, t *domain.TripTemplate) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE trip_templates
		SET route_id = $2, vehicle_id = $3, name = $4,
		    departure_time = NULLIF($5, '')::time,
		    active = $6, updated_at = now()
		WHERE id = $1
	`, t.ID, t.RouteID, t.VehicleID, t.Name, t.DepartureTime, t.Active)
	return err
}
