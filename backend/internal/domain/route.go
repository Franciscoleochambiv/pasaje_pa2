package domain

// Route representa una ruta (ej. Arequipa - Cusco).
type Route struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Active       bool    `json:"active"`
	PricePerSeat float64 `json:"price_per_seat"`
}

// Stop representa una parada en una ruta.
type Stop struct {
	ID       int64  `json:"id"`
	RouteID  int64  `json:"route_id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Position int    `json:"position"`
}

// RouteSegment representa un segmento con precio entre dos paradas.
type RouteSegment struct {
	ID             int64   `json:"id"`
	RouteID        int64   `json:"route_id"`
	OriginStopID   int64   `json:"origin_stop_id"`
	DestStopID     int64   `json:"dest_stop_id"`
	Price          float64 `json:"price"`
	OriginStopName string  `json:"origin_stop_name,omitempty"`
	DestStopName   string  `json:"dest_stop_name,omitempty"`
}
