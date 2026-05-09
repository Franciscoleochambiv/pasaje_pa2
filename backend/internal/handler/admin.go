package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

// AdminHandler maneja los endpoints /api/admin/*.
type AdminHandler struct {
	Admin *service.AdminService
}

// --- Rutas ---

// CreateRoute POST /api/admin/routes
func (h *AdminHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string  `json:"name"`
		Code         string  `json:"code"`
		PricePerSeat float64 `json:"price_per_seat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" || body.Code == "" {
		writeJSONError(w, http.StatusBadRequest, "name y code son requeridos")
		return
	}
	route := &domain.Route{Name: body.Name, Code: body.Code, PricePerSeat: body.PricePerSeat}
	id, err := h.Admin.CreateRoute(r.Context(), route)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateRoute PUT /api/admin/routes/{id}
func (h *AdminHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name         string  `json:"name"`
		Code         string  `json:"code"`
		Active       *bool   `json:"active"`
		PricePerSeat float64 `json:"price_per_seat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	route := &domain.Route{ID: id, Name: body.Name, Code: body.Code, Active: active, PricePerSeat: body.PricePerSeat}
	if err := h.Admin.UpdateRoute(r.Context(), route); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteRoute DELETE /api/admin/routes/{id}
func (h *AdminHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.DeactivateRoute(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// --- Vehículos ---

// ListVehicles GET /api/admin/vehicles
func (h *AdminHandler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	list, err := h.Admin.ListVehicles(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Vehicle{}
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateVehicle POST /api/admin/vehicles
func (h *AdminHandler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Plate     string `json:"plate"`
		Name      string `json:"name"`
		Capacity  int    `json:"capacity"`
		SeatCount int    `json:"seat_count"`
		LayoutID  int64  `json:"layout_id"`
		Seats     []struct {
			Label    string `json:"label"`
			Position int    `json:"position"`
		} `json:"seats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Plate == "" || body.Capacity <= 0 {
		writeJSONError(w, http.StatusBadRequest, "plate y capacity son requeridos")
		return
	}
	var namePtr *string
	if body.Name != "" {
		namePtr = &body.Name
	}
	vehicle := &domain.Vehicle{Plate: body.Plate, Name: namePtr, Capacity: body.Capacity}
	var seats []domain.VehicleSeat
	if body.LayoutID == 0 {
		// Generar asientos secuenciales 1..N si no vinieron explícitos.
		if len(body.Seats) == 0 && body.SeatCount > 0 {
			for i := 1; i <= body.SeatCount; i++ {
				seats = append(seats, domain.VehicleSeat{
					Label: strconv.Itoa(i), Position: i, Floor: 1, RowNum: ((i - 1) / 4) + 1, ColNum: ((i - 1) % 4) + 1, SeatType: "regular",
				})
			}
		} else {
			for _, s := range body.Seats {
				seats = append(seats, domain.VehicleSeat{Label: s.Label, Position: s.Position})
			}
		}
	}
	id, err := h.Admin.CreateVehicle(r.Context(), vehicle, seats, body.LayoutID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateVehicle PUT /api/admin/vehicles/{id}
func (h *AdminHandler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Plate      string `json:"plate"`
		Name       string `json:"name"`
		Capacity   int    `json:"capacity"`
		Active     *bool  `json:"active"`
		Floors     int    `json:"floors"`
		LayoutCols int    `json:"layout_cols"`
		SeatType   string `json:"seat_type"`
		LayoutID   int64  `json:"layout_id"` // 0 = no cambiar; >0 = re-asignar plantilla y re-clonar
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	var namePtr *string
	if body.Name != "" {
		namePtr = &body.Name
	}
	vehicle := &domain.Vehicle{
		ID: id, Plate: body.Plate, Name: namePtr, Capacity: body.Capacity, Active: active,
		Floors: body.Floors, LayoutCols: body.LayoutCols, SeatType: body.SeatType,
	}
	if err := h.Admin.UpdateVehicle(r.Context(), vehicle, body.LayoutID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ListVehicleSeats GET /api/admin/vehicles/{id}/seats
func (h *AdminHandler) ListVehicleSeats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.Admin.ListVehicleSeats(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.VehicleSeat{}
	}
	writeJSON(w, http.StatusOK, list)
}

// DeleteVehicle DELETE /api/admin/vehicles/{id}
func (h *AdminHandler) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.DeactivateVehicle(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// --- Plantillas de viaje ---

// ListTripTemplates GET /api/admin/trip-templates
func (h *AdminHandler) ListTripTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := h.Admin.ListTripTemplates(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.TripTemplate{}
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateTripTemplate POST /api/admin/trip-templates
func (h *AdminHandler) CreateTripTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RouteID       int64  `json:"route_id"`
		VehicleID     int64  `json:"vehicle_id"`
		Name          string `json:"name"`
		DepartureTime string `json:"departure_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.RouteID <= 0 || body.VehicleID <= 0 || body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "route_id, vehicle_id y name son requeridos")
		return
	}
	t := &domain.TripTemplate{
		RouteID:       body.RouteID,
		VehicleID:     body.VehicleID,
		Name:          body.Name,
		DepartureTime: body.DepartureTime,
	}
	id, err := h.Admin.CreateTripTemplate(r.Context(), t)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateTripTemplate PUT /api/admin/trip-templates/{id}
func (h *AdminHandler) UpdateTripTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		RouteID       int64  `json:"route_id"`
		VehicleID     int64  `json:"vehicle_id"`
		Name          string `json:"name"`
		DepartureTime string `json:"departure_time"`
		Active        *bool  `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	t := &domain.TripTemplate{
		ID:            id,
		RouteID:       body.RouteID,
		VehicleID:     body.VehicleID,
		Name:          body.Name,
		DepartureTime: body.DepartureTime,
		Active:        active,
	}
	if err := h.Admin.UpdateTripTemplate(r.Context(), t); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteTripTemplate DELETE /api/admin/trip-templates/{id}
func (h *AdminHandler) DeleteTripTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	t := &domain.TripTemplate{ID: id, Active: false}
	if err := h.Admin.UpdateTripTemplate(r.Context(), t); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// --- Instancias de viaje ---

// ListTripInstances GET /api/admin/trip-instances
func (h *AdminHandler) ListTripInstances(w http.ResponseWriter, r *http.Request) {
	var from time.Time
	if q := r.URL.Query().Get("from"); q != "" {
		from, _ = time.Parse("2006-01-02", q)
	}
	list, err := h.Admin.ListTripInstances(r.Context(), from)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.TripInstanceAdmin{}
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateTripInstance POST /api/admin/trip-instances
func (h *AdminHandler) CreateTripInstance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TripTemplateID int64  `json:"trip_template_id"`
		DepartureAt    string `json:"departure_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.TripTemplateID <= 0 || body.DepartureAt == "" {
		writeJSONError(w, http.StatusBadRequest, "trip_template_id y departure_at son requeridos")
		return
	}
	departureAt, err := time.Parse(time.RFC3339, body.DepartureAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "departure_at debe ser RFC3339 (ej. 2025-01-15T08:00:00Z)")
		return
	}
	id, err := h.Admin.CreateTripInstance(r.Context(), body.TripTemplateID, departureAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateTripInstance PUT /api/admin/trip-instances/{id}
func (h *AdminHandler) UpdateTripInstance(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Status      string `json:"status"`
		DepartureAt string `json:"departure_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Status != "" {
		if err := h.Admin.UpdateTripInstanceStatus(r.Context(), id, body.Status); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if body.DepartureAt != "" {
		departureAt, err := time.Parse(time.RFC3339, body.DepartureAt)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "departure_at debe ser RFC3339")
			return
		}
		if err := h.Admin.UpdateTripInstanceDeparture(r.Context(), id, departureAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteTripInstance DELETE /api/admin/trip-instances/{id}
func (h *AdminHandler) DeleteTripInstance(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.UpdateTripInstanceStatus(r.Context(), id, "cancelled"); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// --- Paradas (Stops) ---

// ListStops GET /api/admin/routes/{id}/stops
func (h *AdminHandler) ListStops(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.Admin.ListStops(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Stop{}
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateStop POST /api/admin/routes/{id}/stops
func (h *AdminHandler) CreateStop(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name     string `json:"name"`
		Code     string `json:"code"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" || body.Code == "" {
		writeJSONError(w, http.StatusBadRequest, "name y code son requeridos")
		return
	}
	// Auto-asignar posición si no viene o es <= 0
	if body.Position <= 0 {
		existing, _ := h.Admin.ListStops(r.Context(), routeID)
		maxPos := 0
		for _, s := range existing {
			if s.Position > maxPos {
				maxPos = s.Position
			}
		}
		body.Position = maxPos + 1
	}
	// Validar que no exista otra parada con la misma posición en esta ruta
	existing, _ := h.Admin.ListStops(r.Context(), routeID)
	for _, s := range existing {
		if s.Position == body.Position {
			writeJSONError(w, http.StatusBadRequest, "ya existe una parada con esa posición en esta ruta")
			return
		}
		if s.Code == body.Code {
			writeJSONError(w, http.StatusBadRequest, "ya existe una parada con ese código en esta ruta")
			return
		}
	}
	stop := &domain.Stop{RouteID: routeID, Name: body.Name, Code: body.Code, Position: body.Position}
	id, err := h.Admin.CreateStop(r.Context(), stop)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateStop PUT /api/admin/stops/{id}
func (h *AdminHandler) UpdateStop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name     string `json:"name"`
		Code     string `json:"code"`
		Position int    `json:"position"`
		RouteID  int64  `json:"route_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" || body.Code == "" {
		writeJSONError(w, http.StatusBadRequest, "name y code son requeridos")
		return
	}
	if body.Position <= 0 {
		writeJSONError(w, http.StatusBadRequest, "posición debe ser mayor a 0")
		return
	}
	// Validar duplicados (excluyendo el stop actual)
	if body.RouteID > 0 {
		existing, _ := h.Admin.ListStops(r.Context(), body.RouteID)
		for _, s := range existing {
			if s.ID == id {
				continue
			}
			if s.Position == body.Position {
				writeJSONError(w, http.StatusBadRequest, "ya existe una parada con esa posición en esta ruta")
				return
			}
			if s.Code == body.Code {
				writeJSONError(w, http.StatusBadRequest, "ya existe una parada con ese código en esta ruta")
				return
			}
		}
	}
	stop := &domain.Stop{ID: id, Name: body.Name, Code: body.Code, Position: body.Position}
	if err := h.Admin.UpdateStop(r.Context(), stop); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteStop DELETE /api/admin/stops/{id}
func (h *AdminHandler) DeleteStop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.DeleteStop(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			writeJSONError(w, http.StatusConflict, "No se puede eliminar: esta parada tiene encomiendas o tramos asociados. Elimina primero las encomiendas y tramos que la usan.")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ReorderStops POST /api/admin/routes/{id}/stops/reorder
// Reasigna posiciones 1,2,3... según el orden actual.
func (h *AdminHandler) ReorderStops(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	stops, err := h.Admin.ListStops(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i, s := range stops {
		s.Position = i + 1
		if err := h.Admin.UpdateStop(r.Context(), &s); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reordered"})
}

// GenerateSegments POST /api/admin/routes/{id}/segments/generate
// Genera tramos consecutivos entre paradas ordenadas por posición.
func (h *AdminHandler) GenerateSegments(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	stops, err := h.Admin.ListStops(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(stops) < 2 {
		writeJSONError(w, http.StatusBadRequest, "se necesitan al menos 2 paradas")
		return
	}
	var created int
	for i := 0; i < len(stops)-1; i++ {
		seg := &domain.RouteSegment{
			RouteID:      routeID,
			OriginStopID: stops[i].ID,
			DestStopID:   stops[i+1].ID,
			Price:        0,
		}
		if _, err := h.Admin.UpsertSegment(r.Context(), seg); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		created++
	}
	writeJSON(w, http.StatusOK, map[string]int{"created": created})
}

// --- Segmentos de ruta ---

// ListSegments GET /api/admin/routes/{id}/segments
func (h *AdminHandler) ListSegments(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.Admin.ListSegments(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.RouteSegment{}
	}
	writeJSON(w, http.StatusOK, list)
}

// UpsertSegment POST /api/admin/routes/{id}/segments
func (h *AdminHandler) UpsertSegment(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		OriginStopID int64   `json:"origin_stop_id"`
		DestStopID   int64   `json:"dest_stop_id"`
		Price        float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.OriginStopID <= 0 || body.DestStopID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "origin_stop_id y dest_stop_id son requeridos")
		return
	}
	if body.OriginStopID == body.DestStopID {
		writeJSONError(w, http.StatusBadRequest, "origen y destino no pueden ser la misma parada")
		return
	}
	if body.Price < 0 {
		writeJSONError(w, http.StatusBadRequest, "precio no puede ser negativo")
		return
	}
	// Validar que los stops pertenezcan a la ruta
	stops, err := h.Admin.ListStops(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	originValid, destValid := false, false
	for _, s := range stops {
		if s.ID == body.OriginStopID {
			originValid = true
		}
		if s.ID == body.DestStopID {
			destValid = true
		}
	}
	if !originValid || !destValid {
		writeJSONError(w, http.StatusBadRequest, "las paradas no pertenecen a esta ruta")
		return
	}
	seg := &domain.RouteSegment{RouteID: routeID, OriginStopID: body.OriginStopID, DestStopID: body.DestStopID, Price: body.Price}
	id, err := h.Admin.UpsertSegment(r.Context(), seg)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// DeleteSegment DELETE /api/admin/segments/{id}
func (h *AdminHandler) DeleteSegment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.DeleteSegment(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Stats ---

// GetStats GET /api/admin/stats
func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Admin.GetDashboardStats(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
