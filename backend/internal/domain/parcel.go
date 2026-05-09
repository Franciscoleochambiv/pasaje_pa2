package domain

import "time"

// Parcel representa una encomienda asociada a un viaje.
type Parcel struct {
	ID               int64   `json:"id"`
	Code             string  `json:"code"`
	TripInstanceID   int64   `json:"trip_instance_id"`
	OriginStopID     int64   `json:"origin_stop_id"`
	DestStopID       int64   `json:"dest_stop_id"`
	OriginStopName   string  `json:"origin_stop_name,omitempty"`
	DestStopName     string  `json:"dest_stop_name,omitempty"`

	// Remitente
	SenderName      string `json:"sender_name"`
	SenderDocType   string `json:"sender_doc_type"`
	SenderDocNumber string `json:"sender_doc_number"`
	SenderPhone     string `json:"sender_phone"`

	// Destinatario
	ReceiverName      string `json:"receiver_name"`
	ReceiverDocType   string `json:"receiver_doc_type"`
	ReceiverDocNumber string `json:"receiver_doc_number"`
	ReceiverPhone     string `json:"receiver_phone"`

	// Carga
	PackageCount int     `json:"package_count"`
	WeightKg     float64 `json:"weight_kg"`
	Description  string  `json:"description"`

	// Pago
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	PaymentMode   string `json:"payment_mode"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	// Facturación
	BillingDocType    string `json:"billing_doc_type"`
	BillingEmail      string `json:"billing_email"`
	BillingRuc        string `json:"billing_ruc"`
	BillingRazonSocial string `json:"billing_razon_social"`
	BillingAddress    string `json:"billing_address"`
	BillingSaleID     *int64 `json:"billing_sale_id"`

	// Estado
	Status       string `json:"status"`
	RegisteredBy *int64 `json:"registered_by"`

	// Datos denormalizados para listados
	RouteName    string `json:"route_name,omitempty"`
	DepartureAt  string `json:"departure_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParcelTracking representa un movimiento en el historial de una encomienda.
type ParcelTracking struct {
	ID        int64     `json:"id"`
	ParcelID  int64     `json:"parcel_id"`
	Status    string    `json:"status"`
	Location  string    `json:"location"`
	Notes     string    `json:"notes"`
	UserID    *int64    `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ParcelPublicInfo es la información pública de tracking.
type ParcelPublicInfo struct {
	Code             string           `json:"code"`
	Status           string           `json:"status"`
	OriginStop       string           `json:"origin_stop"`
	DestStop         string           `json:"dest_stop"`
	ReceiverName     string           `json:"receiver_name"`
	PackageCount     int              `json:"package_count"`
	WeightKg         float64          `json:"weight_kg"`
	Description      string           `json:"description"`
	PaymentMode      string           `json:"payment_mode"`
	PaymentStatus    string           `json:"payment_status"`
	CreatedAt        time.Time        `json:"created_at"`
	Tracking         []ParcelTracking `json:"tracking"`
}

// ParcelStatusLabels etiquetas legibles para estados.
var ParcelStatusLabels = map[string]string{
	"registered":       "Registrada",
	"boarded":          "Embarcada",
	"in_transit":       "En tránsito",
	"arrived":          "Llegó a destino",
	"ready_for_pickup": "Lista para recoger",
	"delivered":        "Entregada",
	"cancelled":        "Anulada",
}

// ValidParcelTransitions define transiciones válidas de estado.
var ValidParcelTransitions = map[string][]string{
	"registered":       {"boarded", "cancelled"},
	"boarded":          {"in_transit", "cancelled"},
	"in_transit":       {"arrived", "cancelled"},
	"arrived":          {"ready_for_pickup", "cancelled"},
	"ready_for_pickup": {"delivered", "cancelled"},
}
