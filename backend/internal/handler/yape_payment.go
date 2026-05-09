package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"

	"github.com/go-chi/chi/v5"
)

// YapePaymentHandler handles Yape direct payment endpoints.
type YapePaymentHandler struct {
	Voucher         *service.VoucherService
	ReservationSvc  *service.ReservationService
	ReservationRepo *repository.ReservationRepository
	Billing         *service.BillingService
	Notifier        *ws.Notifier
	Email           *service.EmailService
	Settings        *repository.SettingsRepository
	Comprobante     *service.ComprobanteRetryService
	HoldMinutes     int
	YapeNumber      string // fallback from env
	WhatsAppPhone   string // fallback from env
}

// VoidReservation POST /api/admin/reservations/{code}/void
// Anula una venta: libera asientos, marca tickets/pagos como voided y broadcastea WS.
func (h *YapePaymentHandler) VoidReservation(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "code es requerido")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Reason) == "" {
		writeJSONError(w, http.StatusBadRequest, "reason es requerido")
		return
	}
	var reviewerID int64
	if claims := UserFromContext(r.Context()); claims != nil {
		reviewerID = claims.UserID
	}
	result, err := h.ReservationRepo.VoidReservation(r.Context(), code, reviewerID, strings.TrimSpace(body.Reason))
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	traceID := fmt.Sprintf("void-res-%s-%d", code, time.Now().UnixNano())
	log.Printf("[void-trace:%s] reservation void committed code=%s reviewer=%d trip_ids=%v billing_sale_ids=%v", traceID, code, reviewerID, result.TripInstanceIDs, result.BillingSaleIDs)
	var actorID *int64
	if reviewerID > 0 {
		actorID = &reviewerID
	}
	writeVoidAudit(r.Context(), h.ReservationRepo.Pool, "reservation", nil, "void.local_committed", actorID, map[string]any{
		"trace_id":         traceID,
		"reservation_code": code,
		"reason":           strings.TrimSpace(body.Reason),
		"trip_ids":         result.TripInstanceIDs,
		"billing_sale_ids": result.BillingSaleIDs,
	})

	// Broadcast WS para que todos los clientes refresquen el seat map.
	if h.Notifier != nil {
		for _, tid := range result.TripInstanceIDs {
			go h.Notifier.NotifySeatChange(context.Background(), tid)
		}
		go h.Notifier.NotifyPaymentStatusChange(code, "voided", body.Reason, nil)
	}

	// Anular en venta (Laravel) — async para no bloquear la respuesta. El estado
	// local ya quedó voided; si la anulación remota falla, queda registrado en logs.
	// Después del DELETE en Laravel, disparamos la comunicación de baja a SUNAT
	// vía go-service (RA factura / RC boleta). Replica el flujo de venta/frontend.
	if h.Billing != nil {
		for _, saleID := range result.BillingSaleIDs {
			id := saleID
			go func() {
				log.Printf("[void-trace:%s] reservation=%s sale=%d tenant void start", traceID, code, id)
				writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.tenant.start", actorID, map[string]any{
					"trace_id":         traceID,
					"reservation_code": code,
					"sale_id":          id,
				})
				status, body, err := h.Billing.VoidSaleDetailed(context.Background(), id)
				if err != nil {
					log.Printf("[void-trace:%s] reservation=%s sale=%d tenant void error status=%d err=%v body=%q", traceID, code, id, status, err, truncateForLog(body))
					writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.tenant.error", actorID, map[string]any{
						"trace_id":         traceID,
						"reservation_code": code,
						"sale_id":          id,
						"status":           status,
						"error":            err.Error(),
						"body":             truncateForLog(body),
					})
					return
				}
				log.Printf("[void-trace:%s] reservation=%s sale=%d tenant void ok status=%d body=%q", traceID, code, id, status, truncateForLog(body))
				writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.tenant.ok", actorID, map[string]any{
					"trace_id":         traceID,
					"reservation_code": code,
					"sale_id":          id,
					"status":           status,
					"body":             truncateForLog(body),
				})
				log.Printf("[void-trace:%s] reservation=%s sale=%d sunat void start", traceID, code, id)
				writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.sunat.start", actorID, map[string]any{
					"trace_id":         traceID,
					"reservation_code": code,
					"sale_id":          id,
				})
				sunatStatus, sunatBody, err := h.Billing.VoidSaleSunatDetailed(context.Background(), id)
				if err != nil {
					log.Printf("[void-trace:%s] reservation=%s sale=%d sunat void error status=%d err=%v body=%q", traceID, code, id, sunatStatus, err, truncateForLog(sunatBody))
					writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.sunat.error", actorID, map[string]any{
						"trace_id":         traceID,
						"reservation_code": code,
						"sale_id":          id,
						"status":           sunatStatus,
						"error":            err.Error(),
						"body":             truncateForLog(sunatBody),
					})
				} else {
					log.Printf("[void-trace:%s] reservation=%s sale=%d sunat void ok status=%d body=%q", traceID, code, id, sunatStatus, truncateForLog(sunatBody))
					writeVoidAudit(context.Background(), h.ReservationRepo.Pool, "reservation", nil, "void.sunat.ok", actorID, map[string]any{
						"trace_id":         traceID,
						"reservation_code": code,
						"sale_id":          id,
						"status":           sunatStatus,
						"body":             truncateForLog(sunatBody),
					})
				}
			}()
		}
	} else {
		log.Printf("[void-trace:%s] reservation=%s billing service disabled; remote void skipped", traceID, code)
		writeVoidAudit(r.Context(), h.ReservationRepo.Pool, "reservation", nil, "void.remote.skipped", actorID, map[string]any{
			"trace_id":         traceID,
			"reservation_code": code,
			"reason":           "billing service disabled",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "voided",
		"reservation_code":  code,
		"trip_instance_ids": result.TripInstanceIDs,
		"billing_sale_ids":  result.BillingSaleIDs,
		"void_trace_id":     traceID,
	})
}

// GetSeatHolder GET /api/admin/trips/{tripId}/seats/{seatInventoryId}/holder
// Returns passenger info for a held/reserved/sold seat — admin-only because it
// exposes PII (name, document, phone, email).
func (h *YapePaymentHandler) GetSeatHolder(w http.ResponseWriter, r *http.Request) {
	tripID, err := strconv.ParseInt(chi.URLParam(r, "tripId"), 10, 64)
	if err != nil || tripID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "tripId invalido")
		return
	}
	seatID, err := strconv.ParseInt(chi.URLParam(r, "seatInventoryId"), 10, 64)
	if err != nil || seatID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "seatInventoryId invalido")
		return
	}
	holder, err := h.ReservationRepo.GetSeatHolder(r.Context(), tripID, seatID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Asiento sin titular")
		return
	}
	writeJSON(w, http.StatusOK, holder)
}

// RetryComprobante POST /api/admin/payments/{id}/retry-comprobante — admin manual retry of
// the billing sale + comprobante email when the synchronous attempt failed.
func (h *YapePaymentHandler) RetryComprobante(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id invalido")
		return
	}
	if h.Comprobante == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Servicio de comprobantes no configurado")
		return
	}
	if err := h.Comprobante.RetryOne(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	payment, _ := h.ReservationRepo.GetPaymentByID(r.Context(), id)
	resp := map[string]any{"status": "ok"}
	if payment != nil {
		resp["billing_sale_id"] = payment.BillingSaleID
		resp["billing_attempts"] = payment.BillingAttempts
		resp["billing_error"] = payment.BillingError
		resp["comprobante_email_sent_at"] = payment.ComprobanteEmailSentAt
	}
	writeJSON(w, http.StatusOK, resp)
}

// getYapeNumber reads from settings DB, falls back to env config.
func (h *YapePaymentHandler) getYapeNumber(ctx context.Context) string {
	if h.Settings != nil {
		if v, err := h.Settings.Get(ctx, "yape_business_number"); err == nil && v != "" {
			return v
		}
	}
	return h.YapeNumber
}

// getWhatsAppPhone reads from settings DB, falls back to env config.
func (h *YapePaymentHandler) getWhatsAppPhone(ctx context.Context) string {
	if h.Settings != nil {
		if v, err := h.Settings.Get(ctx, "whatsapp_phone"); err == nil && v != "" {
			return v
		}
	}
	return h.WhatsAppPhone
}

// GetYapeConfig GET /api/payment/yape-config — returns Yape business number and WhatsApp phone.
func (h *YapePaymentHandler) GetYapeConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"yape_number":    h.getYapeNumber(r.Context()),
		"whatsapp_phone": h.getWhatsAppPhone(r.Context()),
	})
}

// CreateYapeDirectPayment POST /api/payment/yape-direct — customer uploads voucher.
func (h *YapePaymentHandler) CreateYapeDirectPayment(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(h.Voucher.MaxSizeMB+2) * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Error procesando formulario: %v", err))
		return
	}

	tripInstanceID, _ := strconv.ParseInt(r.FormValue("trip_instance_id"), 10, 64)
	amount, _ := strconv.Atoi(r.FormValue("amount"))
	seatsStr := r.FormValue("seats")
	passengerName := r.FormValue("passenger_name")
	passengerDocType := r.FormValue("passenger_doc_type")
	passengerDocNumber := r.FormValue("passenger_doc_number")
	passengerEmail := r.FormValue("passenger_email")
	passengerPhone := r.FormValue("passenger_phone")
	passengerAddress := r.FormValue("passenger_address")
	documentType := r.FormValue("document_type")
	routeName := r.FormValue("route_name")
	seatLabelsStr := r.FormValue("seat_labels")
	pricePerSeat, _ := strconv.ParseFloat(r.FormValue("price_per_seat"), 64)

	var seats []int64
	for _, s := range strings.Split(seatsStr, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			seats = append(seats, id)
		}
	}

	var seatLabels []string
	if seatLabelsStr != "" {
		seatLabels = strings.Split(seatLabelsStr, ",")
	}

	if tripInstanceID <= 0 || len(seats) == 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id y seats son requeridos")
		return
	}
	if amount <= 0 {
		writeJSONError(w, http.StatusBadRequest, "amount debe ser mayor a 0")
		return
	}
	if passengerName == "" || passengerDocNumber == "" || passengerEmail == "" {
		writeJSONError(w, http.StatusBadRequest, "nombre, documento y email son requeridos")
		return
	}

	file, header, err := r.FormFile("voucher")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "voucher (imagen) es requerido")
		return
	}
	defer file.Close()

	if err := h.Voucher.ValidateImage(header); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Step 1: Create reservation with extended hold
	holdDuration := time.Duration(h.HoldMinutes) * time.Minute
	reserveInput := &service.CreateReservationInput{
		TripInstanceID: tripInstanceID,
		Seats:          seats,
		HoldDuration:   holdDuration,
	}
	reserveOut, err := h.ReservationSvc.CreateReservation(r.Context(), reserveInput)
	if err != nil {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Error al reservar asientos: %v", err))
		return
	}

	// Broadcast seat change
	if h.Notifier != nil {
		go h.Notifier.NotifySeatChange(context.Background(), tripInstanceID)
	}

	// Step 2: Save voucher image
	ext := service.ExtFromHeader(header)
	file.Seek(0, io.SeekStart)
	voucherPath, err := h.Voucher.SaveVoucher(file, reserveOut.Code, ext)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Error guardando voucher: %v", err))
		return
	}

	// Step 3: Set reservation to pending_verification
	if err := h.ReservationRepo.UpdateReservationStatus(r.Context(), reserveOut.Code, "pending_verification"); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Error actualizando reserva: %v", err))
		return
	}

	// Step 4: Create payment record
	if documentType == "" {
		documentType = "boleta"
	}
	payment := &domain.Payment{
		ReservationID:      &reserveOut.ReservationID,
		AmountCents:        int64(amount),
		Currency:           "PEN",
		Method:             "yape_directo",
		Status:             "pending_verification",
		VoucherPath:        &voucherPath,
		PassengerName:      passengerName,
		PassengerDocType:   passengerDocType,
		PassengerDocNumber: passengerDocNumber,
		PassengerEmail:     passengerEmail,
		PassengerPhone:     passengerPhone,
		PassengerAddress:   passengerAddress,
		DocumentType:       documentType,
		RouteName:          routeName,
		SeatLabels:         seatLabels,
		TripInstanceID:     &tripInstanceID,
		PricePerSeat:       pricePerSeat,
	}

	_, err = h.ReservationRepo.CreateYapePayment(r.Context(), payment)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Error creando pago: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reservation_code": reserveOut.Code,
		"payment_status":   "pending_verification",
		"hold_expires_at":  reserveOut.ExpiresAt,
		"yape_number":      h.getYapeNumber(r.Context()),
		"whatsapp_phone":   h.getWhatsAppPhone(r.Context()),
	})
}

// GetPaymentStatus GET /api/payment/yape-direct/{code}/status
func (h *YapePaymentHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "code es requerido")
		return
	}

	payment, err := h.ReservationRepo.GetPaymentByReservationCode(r.Context(), code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Pago no encontrado")
		return
	}

	reservation, err := h.ReservationSvc.GetReservation(r.Context(), code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Reserva no encontrada")
		return
	}

	resp := map[string]any{
		"payment_status":     payment.Status,
		"reservation_status": reservation.Status,
	}
	if payment.RejectionReason != nil {
		resp["rejection_reason"] = *payment.RejectionReason
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListPendingVouchers GET /api/admin/vouchers/pending
func (h *YapePaymentHandler) ListPendingVouchers(w http.ResponseWriter, r *http.Request) {
	vouchers, err := h.ReservationRepo.GetPendingVouchers(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	if vouchers == nil {
		vouchers = []domain.VoucherReview{}
	}
	writeJSON(w, http.StatusOK, vouchers)
}

// ServeVoucherImage GET /api/admin/vouchers/{id}/image
func (h *YapePaymentHandler) ServeVoucherImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	payment, err := h.ReservationRepo.GetPaymentByID(r.Context(), id)
	if err != nil || payment.VoucherPath == nil {
		http.Error(w, "voucher not found", http.StatusNotFound)
		return
	}

	fullPath := h.Voucher.GetFullPath(*payment.VoucherPath)
	http.ServeFile(w, r, fullPath)
}

// ApproveVoucher POST /api/admin/vouchers/{id}/approve
func (h *YapePaymentHandler) ApproveVoucher(w http.ResponseWriter, r *http.Request) {
	paymentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || paymentID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id invalido")
		return
	}

	var reviewerID int64
	if claims := UserFromContext(r.Context()); claims != nil {
		reviewerID = claims.UserID
	}

	payment, err := h.ReservationRepo.GetPaymentByID(r.Context(), paymentID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Pago no encontrado")
		return
	}
	if payment.Status != "pending_verification" {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Pago ya procesado (estado: %s)", payment.Status))
		return
	}

	ok, err := h.ReservationRepo.ApprovePayment(r.Context(), paymentID, reviewerID)
	if err != nil || !ok {
		writeJSONError(w, http.StatusConflict, "No se pudo aprobar (ya procesado o error)")
		return
	}

	if payment.ReservationID == nil {
		writeJSONError(w, http.StatusInternalServerError, "pago sin reserva asociada")
		return
	}

	// Get reservation code
	reservation, err := h.ReservationRepo.GetByID(r.Context(), *payment.ReservationID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "no se encontro la reserva")
		return
	}

	// Confirm reservation (creates tickets, marks seats as sold)
	confirmInput := &service.ConfirmReservationInput{
		PaymentMethod:      "yape_directo",
		PaymentReference:   fmt.Sprintf("voucher-payment-%d", paymentID),
		PassengerName:      payment.PassengerName,
		PassengerDocType:   payment.PassengerDocType,
		PassengerDocNumber: payment.PassengerDocNumber,
		PassengerEmail:     payment.PassengerEmail,
		PassengerPhone:     payment.PassengerPhone,
		PassengerAddress:   payment.PassengerAddress,
		DocumentType:       payment.DocumentType,
		RouteName:          payment.RouteName,
		SeatLabels:         payment.SeatLabels,
		PricePerSeat:       payment.PricePerSeat,
	}
	confirmOut, err := h.ReservationSvc.ConfirmReservation(r.Context(), reservation.Code, confirmInput)
	if err != nil {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Error al confirmar reserva: %v", err))
		return
	}

	// Broadcast seat changes
	tripIDs, _ := h.ReservationRepo.GetTripInstanceIDsForPayment(r.Context(), paymentID)
	if h.Notifier != nil {
		for _, tripID := range tripIDs {
			go h.Notifier.NotifySeatChange(context.Background(), tripID)
		}
	}

	// Broadcast payment status INMEDIATAMENTE: el cliente que espera la confirmación
	// no debe quedarse colgado por el tiempo que tarda createBillingSale (puede tardar
	// segundos por reintentos al tenant + go-service). El billing_sale_id se entrega
	// por email cuando esté listo; el cliente sólo necesita saber "aprobado" ya.
	if h.Notifier != nil {
		h.Notifier.NotifyPaymentStatusChange(reservation.Code, "approved", "", map[string]any{
			"ticket_codes": confirmOut.TicketCodes,
		})
	}

	// Create billing sale + email vía ComprobanteRetryService.RetryOne para que toda
	// la lógica viva en un solo lugar (evita doble emisión cuando el worker periódico
	// también escanea la cola de pendientes). Goroutine con context.Background() para
	// que no se cancele si el admin cierra la pestaña antes de que termine.
	if h.Comprobante != nil && payment.DocumentType != "" {
		go func() {
			if err := h.Comprobante.RetryOne(context.Background(), paymentID); err != nil {
				fmt.Printf("payment %d: comprobante: %v\n", paymentID, err)
			}
		}()
	}

	resp := map[string]any{
		"status":           "approved",
		"reservation_code": reservation.Code,
		"ticket_codes":     confirmOut.TicketCodes,
	}
	writeJSON(w, http.StatusOK, resp)
}

// RejectVoucher POST /api/admin/vouchers/{id}/reject
func (h *YapePaymentHandler) RejectVoucher(w http.ResponseWriter, r *http.Request) {
	paymentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || paymentID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id invalido")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Reason == "" {
		writeJSONError(w, http.StatusBadRequest, "reason es requerido")
		return
	}

	var reviewerID int64
	if claims := UserFromContext(r.Context()); claims != nil {
		reviewerID = claims.UserID
	}

	// Get data for broadcast before rejecting
	tripIDs, _ := h.ReservationRepo.GetTripInstanceIDsForPayment(r.Context(), paymentID)
	payment, _ := h.ReservationRepo.GetPaymentByID(r.Context(), paymentID)
	var reservationCode string
	if payment != nil && payment.ReservationID != nil {
		if res, err := h.ReservationRepo.GetByID(r.Context(), *payment.ReservationID); err == nil {
			reservationCode = res.Code
		}
	}

	err = h.ReservationRepo.RejectPayment(r.Context(), paymentID, reviewerID, body.Reason)
	if err != nil {
		writeJSONError(w, http.StatusConflict, fmt.Sprintf("Error: %v", err))
		return
	}

	if h.Notifier != nil {
		for _, tripID := range tripIDs {
			go h.Notifier.NotifySeatChange(context.Background(), tripID)
		}
		if reservationCode != "" {
			go h.Notifier.NotifyPaymentStatusChange(reservationCode, "rejected", body.Reason, nil)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

