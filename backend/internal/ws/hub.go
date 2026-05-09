package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WriteWait is the time allowed to write a message to the peer.
	WriteWait = 10 * time.Second

	// PongWait is the time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second

	// PingPeriod sends pings to peer with this period. Must be less than PongWait.
	PingPeriod = 30 * time.Second
)

// safeConn wraps a websocket.Conn with a write mutex for concurrent safety.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func newSafeConn(conn *websocket.Conn) *safeConn {
	return &safeConn{conn: conn}
}

func (sc *safeConn) WriteMessage(messageType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	return sc.conn.WriteMessage(messageType, data)
}

func (sc *safeConn) WritePing() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	return sc.conn.WriteMessage(websocket.PingMessage, nil)
}

func (sc *safeConn) Close() {
	sc.conn.Close()
}

// Hub manages WebSocket connections grouped by trip_instance_id and reservation_code.
type Hub struct {
	mu               sync.RWMutex
	rooms            map[int64]map[*safeConn]bool  // trip_instance_id -> connections
	reservationRooms map[string]map[*safeConn]bool // reservation_code -> connections
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms:            make(map[int64]map[*safeConn]bool),
		reservationRooms: make(map[string]map[*safeConn]bool),
	}
}

// Subscribe registers a connection for a trip. Returns the safeConn wrapper.
func (h *Hub) Subscribe(tripID int64, conn *websocket.Conn) *safeConn {
	sc := newSafeConn(conn)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[tripID] == nil {
		h.rooms[tripID] = make(map[*safeConn]bool)
	}
	h.rooms[tripID][sc] = true
	log.Printf("ws: client subscribed to trip %d (%d clients)", tripID, len(h.rooms[tripID]))
	return sc
}

// Unsubscribe removes a connection from a trip and closes it.
func (h *Hub) Unsubscribe(tripID int64, sc *safeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[tripID] != nil {
		delete(h.rooms[tripID], sc)
		if len(h.rooms[tripID]) == 0 {
			delete(h.rooms, tripID)
		}
	}
	sc.Close()
}

// Broadcast sends a message to all clients watching a specific trip.
func (h *Hub) Broadcast(tripID int64, message []byte) {
	h.mu.RLock()
	clients := h.rooms[tripID]
	conns := make([]*safeConn, 0, len(clients))
	for sc := range clients {
		conns = append(conns, sc)
	}
	h.mu.RUnlock()

	for _, sc := range conns {
		if err := sc.WriteMessage(websocket.TextMessage, message); err != nil {
			sc.Close()
		}
	}
}

// SubscribeReservation registers a connection for a reservation code. Returns the safeConn wrapper.
func (h *Hub) SubscribeReservation(code string, conn *websocket.Conn) *safeConn {
	sc := newSafeConn(conn)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.reservationRooms[code] == nil {
		h.reservationRooms[code] = make(map[*safeConn]bool)
	}
	h.reservationRooms[code][sc] = true
	log.Printf("ws: client subscribed to reservation %s (%d clients)", code, len(h.reservationRooms[code]))
	return sc
}

// UnsubscribeReservation removes a connection from a reservation room and closes it.
func (h *Hub) UnsubscribeReservation(code string, sc *safeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.reservationRooms[code] != nil {
		delete(h.reservationRooms[code], sc)
		if len(h.reservationRooms[code]) == 0 {
			delete(h.reservationRooms, code)
		}
	}
	sc.Close()
}

// BroadcastReservation sends a message to all clients watching a specific reservation.
func (h *Hub) BroadcastReservation(code string, message []byte) {
	h.mu.RLock()
	clients := h.reservationRooms[code]
	conns := make([]*safeConn, 0, len(clients))
	for sc := range clients {
		conns = append(conns, sc)
	}
	h.mu.RUnlock()

	for _, sc := range conns {
		if err := sc.WriteMessage(websocket.TextMessage, message); err != nil {
			sc.Close()
		}
	}
}
