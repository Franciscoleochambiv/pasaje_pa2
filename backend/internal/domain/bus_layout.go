package domain

import "time"

// BusLayout es la plantilla reutilizable de la geometría interna de un bus.
type BusLayout struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Brand           string    `json:"brand"`
	Model           string    `json:"model"`
	Description     string    `json:"description"`
	Floors          int       `json:"floors"`
	LayoutCols      int       `json:"layout_cols"`
	SeatType        string    `json:"seat_type"`
	PreviewImageURL string    `json:"preview_image_url"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BusLayoutSeat es un asiento dentro de la plantilla.
type BusLayoutSeat struct {
	ID       int64  `json:"id"`
	LayoutID int64  `json:"layout_id"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	Floor    int    `json:"floor"`
	RowNum   int    `json:"row_num"`
	ColNum   int    `json:"col_num"`
	SeatType string `json:"seat_type"`
}

// BusLayoutElement es un elemento decorativo (no-asiento) de la plantilla:
// pasillo, escalera, TV, baño, conductor, ícono Yape, mujer/hombre, cama, pantalla, label libre.
type BusLayoutElement struct {
	ID       int64  `json:"id"`
	LayoutID int64  `json:"layout_id"`
	Floor    int    `json:"floor"`
	RowNum   int    `json:"row_num"`
	ColNum   int    `json:"col_num"`
	Kind     string `json:"kind"`
	Text     string `json:"text"`
}

// BusLayoutFull combina la plantilla con sus asientos y elementos para edición y vista previa.
type BusLayoutFull struct {
	Layout   BusLayout          `json:"layout"`
	Seats    []BusLayoutSeat    `json:"seats"`
	Elements []BusLayoutElement `json:"elements"`
}

// VehicleLayoutElement es la copia clonada al vehículo concreto.
// El cliente final ve estos elementos al elegir asiento.
type VehicleLayoutElement struct {
	ID        int64  `json:"id"`
	VehicleID int64  `json:"vehicle_id"`
	Floor     int    `json:"floor"`
	RowNum    int    `json:"row_num"`
	ColNum    int    `json:"col_num"`
	Kind      string `json:"kind"`
	Text      string `json:"text"`
}
