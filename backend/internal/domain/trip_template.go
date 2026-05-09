package domain

import "time"

// TripTemplate es la plantilla de un viaje (ruta + vehículo + horario).
type TripTemplate struct {
	ID            int64     `json:"id"`
	RouteID       int64     `json:"route_id"`
	VehicleID     int64     `json:"vehicle_id"`
	Name          string    `json:"name"`
	DepartureTime string    `json:"departure_time"` // HH:MM
	Active        bool      `json:"active"`
	RouteName     string    `json:"route_name,omitempty"`
	VehicleName   *string   `json:"vehicle_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
