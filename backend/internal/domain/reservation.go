package domain

import "time"

// Reservation representa una reserva de asientos.
type Reservation struct {
	ID             int64     `json:"id"`
	Code           string    `json:"code"`
	UserID         *int64    `json:"user_id,omitempty"`
	AgencyID       *int64    `json:"agency_id,omitempty"`
	SalesChannelID int64     `json:"sales_channel_id"`
	Status         string    `json:"status"` // pending | confirmed | expired | cancelled
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ContactName      string    `json:"contact_name,omitempty"`
	ContactDocNumber string    `json:"contact_doc_number,omitempty"`
	ContactPhone     string    `json:"contact_phone,omitempty"`
}

// ReservationItem es un asiento reservado dentro de una reserva.
type ReservationItem struct {
	ID                  int64     `json:"id"`
	ReservationID       int64     `json:"reservation_id"`
	TripSeatInventoryID int64     `json:"trip_seat_inventory_id"`
	CreatedAt           time.Time `json:"created_at"`
}

// Ticket representa un boleto emitido tras confirmar la reserva.
type Ticket struct {
	ID                  int64     `json:"id"`
	Code                string    `json:"code"`
	ReservationID       *int64    `json:"reservation_id,omitempty"`
	TripInstanceID      int64     `json:"trip_instance_id"`
	TripSeatInventoryID int64     `json:"trip_seat_inventory_id"`
	UserID              *int64    `json:"user_id,omitempty"`
	AgencyID            *int64    `json:"agency_id,omitempty"`
	SalesChannelID      int64     `json:"sales_channel_id"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Payment representa un pago asociado a un ticket o reserva.
type Payment struct {
	ID              int64      `json:"id"`
	TicketID        *int64     `json:"ticket_id,omitempty"`
	ReservationID   *int64     `json:"reservation_id,omitempty"`
	AmountCents     int64      `json:"amount_cents"`
	Currency        string     `json:"currency"`
	Method          string     `json:"method"`
	Reference       *string    `json:"reference,omitempty"`
	Status          string     `json:"status"`
	VoucherPath     *string    `json:"voucher_path,omitempty"`
	ReviewedBy      *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Passenger data stored on payment for Yape direct flow
	PassengerName      string   `json:"passenger_name,omitempty"`
	PassengerDocType   string   `json:"passenger_doc_type,omitempty"`
	PassengerDocNumber string   `json:"passenger_doc_number,omitempty"`
	PassengerEmail     string   `json:"passenger_email,omitempty"`
	PassengerPhone     string   `json:"passenger_phone,omitempty"`
	PassengerAddress   string   `json:"passenger_address,omitempty"`
	DocumentType       string   `json:"document_type,omitempty"`
	RouteName          string   `json:"route_name,omitempty"`
	SeatLabels         []string `json:"seat_labels,omitempty"`
	TripInstanceID     *int64   `json:"trip_instance_id,omitempty"`
	PricePerSeat       float64  `json:"price_per_seat,omitempty"`

	// Billing follow-up state — populated after the comprobante is emitted.
	BillingSaleID           *int       `json:"billing_sale_id,omitempty"`
	BillingError            *string    `json:"billing_error,omitempty"`
	BillingAttempts         int        `json:"billing_attempts,omitempty"`
	ComprobanteEmailSentAt  *time.Time `json:"comprobante_email_sent_at,omitempty"`
}

// ReservationPublicInfo es la vista enriquecida que se devuelve al consultar
// una reserva por código (consulta pública de cliente). Incluye datos del
// pasajero, asientos con etiqueta, viaje (ruta + fecha + paraderos) y tickets
// emitidos para que el cliente pueda ver toda la información sin login admin.
type ReservationPublicInfo struct {
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	// Pasajero (de payments)
	PassengerName      string `json:"passenger_name,omitempty"`
	PassengerDocType   string `json:"passenger_doc_type,omitempty"`
	PassengerDocNumber string `json:"passenger_doc_number,omitempty"`
	PassengerEmail     string `json:"passenger_email,omitempty"`
	PassengerPhone     string `json:"passenger_phone,omitempty"`

	// Comprobante / pago
	DocumentType  string  `json:"document_type,omitempty"`
	PaymentMethod string  `json:"payment_method,omitempty"`
	PaymentStatus string  `json:"payment_status,omitempty"`
	BillingSaleID *int    `json:"billing_sale_id,omitempty"`
	TotalAmount   float64 `json:"total_amount,omitempty"`

	// Viaje
	RouteName      string     `json:"route_name,omitempty"`
	OriginStop     string     `json:"origin_stop,omitempty"`
	DestStop       string     `json:"dest_stop,omitempty"`
	DepartureAt    *time.Time `json:"departure_at,omitempty"`
	TripInstanceID *int64     `json:"trip_instance_id,omitempty"`

	// Asientos + tickets
	Items []ReservationPublicItem `json:"items"`
}

// ReservationPublicItem es un asiento de la reserva con su etiqueta y el
// código de boleto emitido (si la reserva fue confirmada).
type ReservationPublicItem struct {
	ID           int64  `json:"id"`
	SeatLabel    string `json:"seat_label"`
	TicketCode   string `json:"ticket_code,omitempty"`
	TicketStatus string `json:"ticket_status,omitempty"`
}

// VoucherReview is a denormalized view for the admin verification dashboard.
type VoucherReview struct {
	PaymentID          int64     `json:"payment_id"`
	ReservationCode    string    `json:"reservation_code"`
	ReservationID      int64     `json:"reservation_id"`
	AmountCents        int64     `json:"amount_cents"`
	PassengerName      string    `json:"passenger_name"`
	PassengerDocType   string    `json:"passenger_doc_type"`
	PassengerDocNumber string    `json:"passenger_doc_number"`
	PassengerEmail     string    `json:"passenger_email"`
	PassengerPhone     string    `json:"passenger_phone"`
	DocumentType       string    `json:"document_type"`
	RouteName          string    `json:"route_name"`
	SeatLabels         []string  `json:"seat_labels"`
	TripInstanceID     *int64    `json:"trip_instance_id"`
	PricePerSeat       float64   `json:"price_per_seat"`
	PassengerAddress   string    `json:"passenger_address"`
	CreatedAt          time.Time `json:"created_at"`
	HoldExpiresAt      time.Time `json:"hold_expires_at"`
}
