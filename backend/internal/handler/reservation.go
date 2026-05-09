package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// ReservationHandler maneja los endpoints /api/reservations.
type ReservationHandler struct {
	Reservations    *service.ReservationService
	ReservationRepo *repository.ReservationRepository
	Notifier        *ws.Notifier
	SettingsRepo    *repository.SettingsRepository
}

// Create POST /api/reservations
func (h *ReservationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TripInstanceID    int64   `json:"trip_instance_id"`
		Seats             []int64 `json:"seats"`
		PassengerName     string  `json:"passenger_name"`
		PassengerDocType  string  `json:"passenger_doc_type"`
		PassengerDocNumber string `json:"passenger_doc_number"`
		PassengerEmail    string  `json:"passenger_email"`
		PassengerPhone    string  `json:"passenger_phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.TripInstanceID <= 0 || len(body.Seats) == 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id y seats son requeridos")
		return
	}

	input := &service.CreateReservationInput{
		TripInstanceID: body.TripInstanceID,
		Seats:          body.Seats,
	}
	out, err := h.Reservations.CreateReservation(r.Context(), input)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)

	// Broadcast seat update via WebSocket
	if h.Notifier != nil {
		go h.Notifier.NotifySeatChange(context.Background(), body.TripInstanceID)
	}
}

// Confirm POST /api/reservations/{code}/confirm
func (h *ReservationHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "código de reserva requerido")
		return
	}

	var body service.ConfirmReservationInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.PaymentMethod == "" {
		writeJSONError(w, http.StatusBadRequest, "payment_method es requerido")
		return
	}

	out, err := h.Reservations.ConfirmReservation(r.Context(), code, &body)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)

	// Broadcast seat update via WebSocket
	if h.Notifier != nil {
		tripIDs, _ := h.Reservations.GetTripInstanceIDsByCode(r.Context(), code)
		for _, tid := range tripIDs {
			go h.Notifier.NotifySeatChange(context.Background(), tid)
		}
	}
}

// GetByCode GET /api/reservations/{code}
func (h *ReservationHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "código de reserva requerido")
		return
	}

	info, err := h.ReservationRepo.GetPublicByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "reserva no encontrada")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// CreateAdmin POST /api/admin/reservations — crea una reserva desde punto de venta.
func (h *ReservationHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TripInstanceID   int64   `json:"trip_instance_id"`
		Seats            []int64 `json:"seats"`
		ContactName      string  `json:"contact_name"`
		ContactDocNumber string  `json:"contact_doc_number"`
		ContactPhone     string  `json:"contact_phone"`
		HoldHours        int     `json:"hold_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.TripInstanceID <= 0 || len(body.Seats) == 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id y seats son requeridos")
		return
	}

	// Determinar horas de retención
	holdHours := body.HoldHours
	if holdHours <= 0 && h.SettingsRepo != nil {
		if v, err := h.SettingsRepo.Get(r.Context(), "pos_hold_hours"); err == nil {
			if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
				holdHours = parsed
			}
		}
	}
	if holdHours <= 0 {
		holdHours = 24
	}

	channelID, err := h.ReservationRepo.GetOrCreatePOSChannel(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error obteniendo canal POS")
		return
	}

	input := &service.CreateReservationInput{
		TripInstanceID:   body.TripInstanceID,
		Seats:            body.Seats,
		HoldDuration:     time.Duration(holdHours) * time.Hour,
		SalesChannelID:   channelID,
		ContactName:      body.ContactName,
		ContactDocNumber: body.ContactDocNumber,
		ContactPhone:     body.ContactPhone,
	}
	out, err := h.Reservations.CreateReservation(r.Context(), input)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)

	if h.Notifier != nil {
		go h.Notifier.NotifySeatChange(context.Background(), body.TripInstanceID)
	}
}

// ListAdmin GET /api/admin/reservations — lista reservas pending por viaje.
func (h *ReservationHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	tripIDStr := r.URL.Query().Get("trip_instance_id")
	if tripIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id es requerido")
		return
	}
	tripID, err := strconv.ParseInt(tripIDStr, 10, 64)
	if err != nil || tripID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id inválido")
		return
	}

	list, err := h.ReservationRepo.ListPendingReservationsByTrip(r.Context(), tripID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []repository.PendingReservationView{}
	}
	writeJSON(w, http.StatusOK, list)
}
