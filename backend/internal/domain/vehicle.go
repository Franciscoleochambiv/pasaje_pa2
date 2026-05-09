package domain

import "time"

// Vehicle representa un vehículo (bus).
type Vehicle struct {
	ID         int64     `json:"id"`
	Plate      string    `json:"plate"`
	Name       *string   `json:"name"`
	Capacity   int       `json:"capacity"`
	Floors     int       `json:"floors"`      // 1 o 2 pisos
	LayoutCols int       `json:"layout_cols"` // columnas por fila (ej: 4 = 2+pasillo+2)
	SeatType   string    `json:"seat_type"`   // regular | semi_cama | cama | suite
	LayoutID   *int64    `json:"layout_id"`   // bus_layout del que se generó (NULL si fue manual)
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VehicleSeat representa un asiento de la plantilla del vehículo.
type VehicleSeat struct {
	ID        int64  `json:"id"`
	VehicleID int64  `json:"vehicle_id"`
	Label     string `json:"label"`
	Position  int    `json:"position"`
	Floor     int    `json:"floor"`     // piso: 1 o 2
	RowNum    int    `json:"row_num"`   // fila desde el frente
	ColNum    int    `json:"col_num"`   // columna (1=izq, 2=izq-centro, 3=der-centro, 4=der)
	SeatType  string `json:"seat_type"` // tipo individual del asiento
}
