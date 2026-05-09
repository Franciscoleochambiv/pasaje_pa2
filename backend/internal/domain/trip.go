package domain

import "time"

// TripInstance es una salida programada en una fecha concreta.
type TripInstance struct {
	ID             int64     `json:"id"`
	TripTemplateID int64     `json:"trip_template_id"`
	RouteID        int64     `json:"route_id"`
	RouteName      string    `json:"route_name"`
	PricePerSeat   float64   `json:"price_per_seat"`
	DepartureAt    time.Time `json:"departure_at"`
	Status         string    `json:"status"`
}

// TripInstanceAdmin es una instancia con datos extendidos para admin.
type TripInstanceAdmin struct {
	ID             int64     `json:"id"`
	TripTemplateID int64     `json:"trip_template_id"`
	RouteID        int64     `json:"route_id"`
	RouteName      string    `json:"route_name"`
	VehicleID      int64     `json:"vehicle_id"`
	VehicleName    *string   `json:"vehicle_name"`
	DepartureAt    time.Time `json:"departure_at"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SeatInfo es un asiento del inventario de un viaje con su estado.
type SeatInfo struct {
	ID            int64      `json:"id"`
	VehicleSeatID int64      `json:"vehicle_seat_id"`
	Label         string     `json:"label"`
	Position      int        `json:"position"`
	Floor         int        `json:"floor"`
	RowNum        int        `json:"row_num"`
	ColNum        int        `json:"col_num"`
	SeatType      string     `json:"seat_type"`
	Status        string     `json:"status"` // available | held | sold | blocked
	HoldExpiresAt *time.Time `json:"hold_expires_at,omitempty"`
}

// TripSeatResponse wraps seat info with vehicle metadata for the frontend diagram.
type TripSeatResponse struct {
	VehicleFloors     int                    `json:"vehicle_floors"`
	VehicleLayoutCols int                    `json:"vehicle_layout_cols"`
	VehicleSeatType   string                 `json:"vehicle_seat_type"`
	VehicleName       string                 `json:"vehicle_name"`
	RouteName         string                 `json:"route_name"`
	PricePerSeat      float64                `json:"price_per_seat"`
	Seats             []SeatInfo             `json:"seats"`
	LayoutElements    []VehicleLayoutElement `json:"layout_elements"`
}
