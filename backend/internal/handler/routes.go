package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// RoutesHandler maneja /api/routes y viajes/asientos.
type RoutesHandler struct {
	Routes *service.RoutesService
}

// ListRoutes GET /api/routes
func (h *RoutesHandler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	list, err := h.Routes.ListRoutes(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Route{}
	}
	writeJSON(w, http.StatusOK, list)
}

// ListTripsByRoute GET /api/routes/{id}/trips
func (h *RoutesHandler) ListTripsByRoute(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id de ruta inválido")
		return
	}
	var fromDate time.Time
	if q := r.URL.Query().Get("from"); q != "" {
		fromDate, _ = time.Parse("2006-01-02", q)
	}
	list, err := h.Routes.ListTripsByRoute(r.Context(), routeID, fromDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.TripInstance{}
	}
	writeJSON(w, http.StatusOK, list)
}

// ListStops GET /api/routes/{id}/stops
func (h *RoutesHandler) ListStops(w http.ResponseWriter, r *http.Request) {
	routeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || routeID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id de ruta inválido")
		return
	}
	list, err := h.Routes.ListStops(r.Context(), routeID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Stop{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GetTripSeats GET /api/trips/{id}/seats
// Devuelve asientos con metadata del vehículo (pisos, layout) para el diagrama visual.
func (h *RoutesHandler) GetTripSeats(w http.ResponseWriter, r *http.Request) {
	tripID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tripID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id de viaje inválido")
		return
	}
	resp, err := h.Routes.GetTripSeatResponse(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "viaje no encontrado")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
