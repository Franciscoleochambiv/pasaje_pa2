package ws

import (
	"context"
	"encoding/json"
	"log"

	"pasaje/backend/internal/service"
)

// Notifier broadcasts seat changes to WebSocket clients.
type Notifier struct {
	Hub    *Hub
	Routes *service.RoutesService
}

// NotifySeatChange broadcasts the updated seat state for a trip.
func (n *Notifier) NotifySeatChange(ctx context.Context, tripInstanceID int64) {
	resp, err := n.Routes.GetTripSeatResponse(ctx, tripInstanceID)
	if err != nil {
		log.Printf("ws notifier: error getting seats for trip %d: %v", tripInstanceID, err)
		return
	}

	data, err := json.Marshal(map[string]any{
		"type":             "seat_update",
		"trip_instance_id": tripInstanceID,
		"data":             resp,
	})
	if err != nil {
		return
	}

	n.Hub.Broadcast(tripInstanceID, data)
	log.Printf("ws: broadcast seat update for trip %d", tripInstanceID)
}

// NotifyParcelUpdate broadcasts a parcel status change to clients watching a parcel code.
func (n *Notifier) NotifyParcelUpdate(parcelCode string, status string) {
	payload := map[string]any{
		"type":        "parcel_status_update",
		"parcel_code": parcelCode,
		"status":      status,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	n.Hub.BroadcastReservation(parcelCode, data) // reutiliza reservationRooms
	log.Printf("ws: broadcast parcel status %s for %s", status, parcelCode)
}

// NotifyPaymentStatusChange broadcasts a payment status change to clients watching a reservation.
func (n *Notifier) NotifyPaymentStatusChange(reservationCode string, status string, rejectionReason string, extra map[string]any) {
	payload := map[string]any{
		"type":             "payment_status_update",
		"reservation_code": reservationCode,
		"status":           status,
	}
	if rejectionReason != "" {
		payload["rejection_reason"] = rejectionReason
	}
	for k, v := range extra {
		payload[k] = v
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	n.Hub.BroadcastReservation(reservationCode, data)
	log.Printf("ws: broadcast payment status %s for reservation %s", status, reservationCode)
}
