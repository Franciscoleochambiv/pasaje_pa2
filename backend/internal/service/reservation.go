package service

import (
	"context"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
)

// ReservationService lógica de negocio de reservas.
type ReservationService struct {
	ReservationRepo *repository.ReservationRepository
}

// CreateReservationInput datos de entrada para crear una reserva.
type CreateReservationInput struct {
	TripInstanceID   int64         `json:"trip_instance_id"`
	Seats            []int64       `json:"seats"`
	HoldDuration     time.Duration // 0 = default 10 minutes
	SalesChannelID   int64         // 0 = auto (web)
	ContactName      string        `json:"contact_name,omitempty"`
	ContactDocNumber string        `json:"contact_doc_number,omitempty"`
	ContactPhone     string        `json:"contact_phone,omitempty"`
}

// CreateReservationOutput resultado de crear una reserva.
type CreateReservationOutput struct {
	ReservationID int64     `json:"reservation_id"`
	Code          string    `json:"code"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// CreateReservation crea una nueva reserva.
func (s *ReservationService) CreateReservation(ctx context.Context, input *CreateReservationInput) (*CreateReservationOutput, error) {
	var channelID int64
	var err error
	if input.SalesChannelID > 0 {
		channelID = input.SalesChannelID
	} else {
		channelID, err = s.ReservationRepo.GetOrCreateWebChannel(ctx)
		if err != nil {
			return nil, err
		}
	}

	code := repository.GenerateReservationCode()
	holdDuration := 10 * time.Minute
	if input.HoldDuration > 0 {
		holdDuration = input.HoldDuration
	}
	expiresAt := time.Now().Add(holdDuration)

	res, err := s.ReservationRepo.CreateReservation(ctx, code, input.TripInstanceID, input.Seats, channelID, expiresAt, input.ContactName, input.ContactDocNumber, input.ContactPhone)
	if err != nil {
		return nil, err
	}

	return &CreateReservationOutput{
		ReservationID: res.ID,
		Code:          res.Code,
		ExpiresAt:     res.ExpiresAt,
	}, nil
}

// ConfirmReservationInput datos de entrada para confirmar una reserva.
type ConfirmReservationInput struct {
	PaymentMethod      string `json:"payment_method"`
	PaymentReference   string `json:"payment_reference"`
	PassengerName      string `json:"passenger_name"`
	PassengerDocType   string `json:"passenger_doc_type"`
	PassengerDocNumber string `json:"passenger_doc_number"`
	PassengerEmail     string `json:"passenger_email"`
	PassengerPhone     string `json:"passenger_phone"`
	PassengerAddress   string `json:"passenger_address"`
	DocumentType       string `json:"document_type"`
	RouteName          string `json:"route_name"`
	SeatLabels         []string `json:"seat_labels"`
	PricePerSeat       float64 `json:"price_per_seat"`
}

// ConfirmReservationOutput resultado de confirmar una reserva.
type ConfirmReservationOutput struct {
	TicketCodes []string `json:"ticket_codes"`
}

// ConfirmReservation confirma una reserva existente.
func (s *ReservationService) ConfirmReservation(ctx context.Context, code string, input *ConfirmReservationInput) (*ConfirmReservationOutput, error) {
	data := &repository.ConfirmData{
		PaymentMethod:      input.PaymentMethod,
		PaymentReference:   input.PaymentReference,
		PassengerName:      input.PassengerName,
		PassengerDocType:   input.PassengerDocType,
		PassengerDocNumber: input.PassengerDocNumber,
		PassengerEmail:     input.PassengerEmail,
		PassengerPhone:     input.PassengerPhone,
		PassengerAddress:   input.PassengerAddress,
		DocumentType:       input.DocumentType,
		RouteName:          input.RouteName,
		SeatLabels:         input.SeatLabels,
		PricePerSeat:       input.PricePerSeat,
	}
	ticketCodes, err := s.ReservationRepo.ConfirmReservation(ctx, code, data)
	if err != nil {
		return nil, err
	}
	return &ConfirmReservationOutput{TicketCodes: ticketCodes}, nil
}

// GetReservation devuelve una reserva por su código.
func (s *ReservationService) GetReservation(ctx context.Context, code string) (*domain.Reservation, error) {
	return s.ReservationRepo.GetByCode(ctx, code)
}

// GetTripInstanceIDsByCode returns the trip instance IDs associated with a reservation.
func (s *ReservationService) GetTripInstanceIDsByCode(ctx context.Context, code string) ([]int64, error) {
	return s.ReservationRepo.GetTripInstanceIDsByCode(ctx, code)
}
