package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"
)

// PaymentHandler handles Culqi payment endpoints.
type PaymentHandler struct {
	PublicKey       string
	Payment         *service.PaymentService
	Reservation     *service.ReservationService
	Billing         *service.BillingService
	Notifier        *ws.Notifier
	Email           *service.EmailService
	ReservationRepo *repository.ReservationRepository
}

// GetConfig GET /api/payment/config — returns the Culqi public key.
func (h *PaymentHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"public_key": h.PublicKey,
	})
}

// CreateCharge POST /api/payment/charge — process Culqi payment and create reservation.
func (h *PaymentHandler) CreateCharge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TokenID            string   `json:"token_id"`
		Amount             int      `json:"amount"` // cents
		Email              string   `json:"email"`
		Description        string   `json:"description"`
		TripInstanceID     int64    `json:"trip_instance_id"`
		ReservationCode    string   `json:"reservation_code"`
		Seats              []int64  `json:"seats"`
		PassengerName      string   `json:"passenger_name"`
		PassengerDocType   string   `json:"passenger_doc_type"`
		PassengerDocNumber string   `json:"passenger_doc_number"`
		PassengerAddress   string   `json:"passenger_address"`
		DocumentType       string   `json:"document_type"` // boleta | factura
		RouteName          string   `json:"route_name"`
		SeatLabels         []string `json:"seat_labels"`
		PricePerSeat       float64  `json:"price_per_seat"`
		PassengerPhone     string   `json:"passenger_phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	if body.TokenID == "" {
		writeJSONError(w, http.StatusBadRequest, "token_id es requerido")
		return
	}
	if body.Amount <= 0 {
		writeJSONError(w, http.StatusBadRequest, "amount debe ser mayor a 0")
		return
	}
	if body.TripInstanceID <= 0 || len(body.Seats) == 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id y seats son requeridos")
		return
	}
	if body.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "email es requerido")
		return
	}

	// Step 1: Create Culqi charge
	chargeReq := &service.CulqiChargeRequest{
		Amount:       body.Amount,
		CurrencyCode: "PEN",
		SourceID:     body.TokenID,
		Email:        body.Email,
		Description:  body.Description,
	}

	charge, err := h.Payment.CreateCharge(r.Context(), chargeReq)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Error en el pago: %v", err))
		return
	}

	// Step 2: Create reservation (hold seats)
	reserveInput := &service.CreateReservationInput{
		TripInstanceID: body.TripInstanceID,
		Seats:          body.Seats,
	}
	reserveOut, err := h.Reservation.CreateReservation(r.Context(), reserveInput)
	if err != nil {
		// Payment succeeded but reservation failed — log the charge ID for manual reconciliation
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Pago exitoso (cargo: %s) pero error al reservar: %v. Contacta soporte.", charge.ID, err))
		return
	}

	// Broadcast seat change
	if h.Notifier != nil {
		go h.Notifier.NotifySeatChange(context.Background(), body.TripInstanceID)
	}

	// Step 3: Confirm reservation immediately
	confirmInput := &service.ConfirmReservationInput{
		PaymentMethod:      "culqi",
		PaymentReference:   charge.ID,
		PassengerName:      body.PassengerName,
		PassengerDocType:   body.PassengerDocType,
		PassengerDocNumber: body.PassengerDocNumber,
		PassengerEmail:     body.Email,
		PassengerPhone:     body.PassengerPhone,
		PassengerAddress:   body.PassengerAddress,
		DocumentType:       body.DocumentType,
		RouteName:          body.RouteName,
		SeatLabels:         body.SeatLabels,
		PricePerSeat:       body.PricePerSeat,
	}
	confirmOut, err := h.Reservation.ConfirmReservation(r.Context(), reserveOut.Code, confirmInput)
	if err != nil {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Pago exitoso (cargo: %s), reserva creada (%s) pero error al confirmar: %v", charge.ID, reserveOut.Code, err))
		return
	}

	// Broadcast seat change again after confirm
	if h.Notifier != nil {
		go h.Notifier.NotifySeatChange(context.Background(), body.TripInstanceID)
	}

	// Step 4: Create billing sale (boleta/factura)
	var billingSaleID int
	billingError := ""

	if body.DocumentType == "" {
		body.DocumentType = "boleta"
	}

	idTdocumento := 3
	if body.DocumentType == "factura" {
		idTdocumento = 1
	} else if body.DocumentType == "pedido" {
		idTdocumento = 2
	}

	pricePerSeat := body.PricePerSeat
	if pricePerSeat <= 0 {
		// Calculate from amount in cents / number of seats
		pricePerSeat = float64(body.Amount) / 100.0 / float64(len(body.Seats))
	}

	var detalle []service.BillingSaleDetail
	for _, label := range body.SeatLabels {
		detalle = append(detalle, service.BillingSaleDetail{
			IDProducto:      1,
			ProdCodpro:      "777", // 777 = código genérico; el tenant usa detvDescripcion en vez de prodNombre
			IDUmedida:       1,
			DetvCantidad:    1,
			DetvPrecio:      pricePerSeat,
			DetvDescripcion: fmt.Sprintf("ASIENTO %s - Pasaje %s", label, body.RouteName),
		})
	}

	if len(detalle) > 0 {
		clienteID, err := h.Billing.FindOrCreateClient(r.Context(), body.PassengerDocType, body.PassengerDocNumber, body.PassengerName, body.PassengerAddress)
		if err != nil {
			billingError = fmt.Sprintf("Error creando cliente: %v", err)
		} else {
			totalAmount := pricePerSeat * float64(len(body.SeatLabels))
			sale := &service.BillingSaleRequest{
				VenSerie:         1,
				VenFecha:         time.Now().Format("2006-01-02"),
				VenCondiciones:   "Contado",
				VenImporteLetras: service.AmountInLettersPEN(totalAmount),
				VenNroguia:       "",
				VenSaldo:         0,
				Observaciones:    fmt.Sprintf("Reserva: %s / Pago Culqi: %s / Doc: %s %s", reserveOut.Code, charge.ID, body.PassengerDocType, body.PassengerDocNumber),
				DireccionFactura: body.PassengerAddress,
				IDCliente:        clienteID,
				IDEmpleado:       nil,
				IDTdocumento:     idTdocumento,
				IDAlmacen:        1,
				IDTipoventa:      2,
				IDCorr:           1,
				IDVendedor:       nil,
				Detalle:          detalle,
			}

			result, err := h.Billing.CreateSale(r.Context(), sale)
			if err != nil {
				billingError = fmt.Sprintf("Error generando comprobante: %v", err)
				if result != nil {
					billingSaleID = result.SaleID
				}
			} else {
				billingSaleID = result.SaleID
			}

			// Persistir billing_sale_id en payments para que la anulación pueda
			// cascadear DELETE /api/venta/{id} al tenant. Sin esto, la venta
			// queda viva en el tenant y la reventa del asiento la duplica.
			if billingSaleID > 0 && h.ReservationRepo != nil {
				if err := h.ReservationRepo.SetReservationBillingSaleIDByCode(r.Context(), reserveOut.Code, billingSaleID); err != nil {
					fmt.Printf("payment culqi %s: error guardando billing_sale_id=%d: %v\n", reserveOut.Code, billingSaleID, err)
				}
			}
		}
	}

	// Send confirmation email (async, don't block response)
	if h.Email != nil && body.Email != "" {
		go func() {
			totalSoles := float64(body.Amount) / 100.0
			err := h.Email.SendTicketEmail(
				body.Email,
				body.PassengerName,
				reserveOut.Code,
				confirmOut.TicketCodes,
				charge.ID,
				totalSoles,
				body.SeatLabels,
				body.RouteName,
				billingSaleID,
				h.Billing.TenantSlug,
			)
			if err != nil {
				fmt.Printf("email: error sending to %s: %v\n", body.Email, err)
			} else {
				fmt.Printf("email: sent confirmation to %s for reservation %s\n", body.Email, reserveOut.Code)
			}
		}()
	}

	resp := map[string]any{
		"charge_id":        charge.ID,
		"reservation_code": reserveOut.Code,
		"ticket_codes":     confirmOut.TicketCodes,
		"billing_sale_id":  billingSaleID,
		"email_sent":       body.Email != "" && h.Email != nil,
	}
	if billingError != "" {
		resp["billing_error"] = billingError
	}

	writeJSON(w, http.StatusOK, resp)
}
