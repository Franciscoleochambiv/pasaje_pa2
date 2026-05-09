package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins in dev
}

// WSHandler handles WebSocket connections for real-time seat updates.
type WSHandler struct {
	Hub    *ws.Hub
	Routes *service.RoutesService
}

// HandleTripSeats handles WS connections for real-time seat updates.
func (h *WSHandler) HandleTripSeats(w http.ResponseWriter, r *http.Request) {
	tripID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tripID <= 0 {
		http.Error(w, "invalid trip id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sc := h.Hub.Subscribe(tripID, conn)
	defer h.Hub.Unsubscribe(tripID, sc)

	// Send initial seat state
	resp, err := h.Routes.GetTripSeatResponse(r.Context(), tripID)
	if err == nil {
		data, _ := json.Marshal(map[string]any{
			"type":             "seat_update",
			"trip_instance_id": tripID,
			"data":             resp,
		})
		sc.WriteMessage(websocket.TextMessage, data)
	}

	// Configure pong handler to extend read deadline
	conn.SetReadDeadline(time.Now().Add(ws.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(ws.PongWait))
		return nil
	})

	// Start ping ticker in background
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(ws.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := sc.WritePing(); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Keep connection alive, read messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	close(done)
}

// HandleReservationStatus handles WS connections for real-time payment status updates.
func (h *WSHandler) HandleReservationStatus(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "invalid reservation code", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sc := h.Hub.SubscribeReservation(code, conn)
	defer h.Hub.UnsubscribeReservation(code, sc)

	// Configure pong handler to extend read deadline
	conn.SetReadDeadline(time.Now().Add(ws.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(ws.PongWait))
		return nil
	})

	// Start ping ticker in background
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(ws.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := sc.WritePing(); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	close(done)
}
