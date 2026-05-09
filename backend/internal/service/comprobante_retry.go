package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
)

// ComprobanteRetryService retries billing-sale emission for approved Yape-direct payments
// whose comprobante was not generated synchronously (transient upstream errors). Once the
// sale is emitted it sends a follow-up email with the PDF/XML/CDR links to the customer.
type ComprobanteRetryService struct {
	Pool            interface{} // unused but kept symmetric with other services
	ReservationRepo *repository.ReservationRepository
	Billing         *BillingService
	Email           *EmailService
	MaxAttempts     int // hard cap; default 8
}

// StartWorker schedules background retries every interval. Cheap query — only scans rows
// matching the partial index added in migration 000014.
func (s *ComprobanteRetryService) StartWorker(ctx context.Context, interval time.Duration) {
	if s.MaxAttempts <= 0 {
		s.MaxAttempts = 8
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		log.Printf("comprobante retry worker: running every %s", interval)
		for {
			select {
			case <-ctx.Done():
				log.Println("comprobante retry worker: stopped")
				return
			case <-ticker.C:
				if err := s.tick(ctx); err != nil {
					log.Printf("comprobante retry worker: %v", err)
				}
			}
		}
	}()
}

func (s *ComprobanteRetryService) tick(ctx context.Context) error {
	if s.ReservationRepo == nil || s.Billing == nil {
		return nil
	}
	ids, err := s.ReservationRepo.ListPaymentsPendingComprobante(ctx, s.MaxAttempts, 20)
	if err != nil {
		return fmt.Errorf("list pending: %w", err)
	}
	for _, id := range ids {
		if err := s.RetryOne(ctx, id); err != nil {
			log.Printf("comprobante retry: payment %d: %v", id, err)
		}
	}
	return nil
}

// RetryOne re-attempts the billing sale for a single payment and, on success, sends the
// comprobante email. Used by the worker AND by the admin "retry" endpoint.
func (s *ComprobanteRetryService) RetryOne(ctx context.Context, paymentID int64) error {
	payment, err := s.ReservationRepo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("load payment: %w", err)
	}
	if payment.BillingSaleID != nil && *payment.BillingSaleID > 0 {
		// Already emitted — just (re)send the email if it never went out.
		if payment.ComprobanteEmailSentAt == nil {
			return s.sendEmail(ctx, payment)
		}
		return nil
	}
	if payment.DocumentType == "" {
		return nil
	}

	saleID, errMsg := s.createSale(ctx, payment)
	attempts := payment.BillingAttempts + 1
	var saleIDPtr *int
	if saleID > 0 {
		saleIDPtr = &saleID
		payment.BillingSaleID = &saleID
	}
	if err := s.ReservationRepo.SetPaymentBillingResult(ctx, paymentID, saleIDPtr, errMsg, attempts); err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	if saleID == 0 {
		return fmt.Errorf("billing failed: %s", errMsg)
	}
	return s.sendEmail(ctx, payment)
}

func (s *ComprobanteRetryService) sendEmail(ctx context.Context, payment *domain.Payment) error {
	if s.Email == nil || payment.PassengerEmail == "" {
		return nil
	}
	saleID := 0
	if payment.BillingSaleID != nil {
		saleID = *payment.BillingSaleID
	}
	reservationCode := ""
	if payment.ReservationID != nil {
		if res, err := s.ReservationRepo.GetByID(ctx, *payment.ReservationID); err == nil {
			reservationCode = res.Code
		}
	}
	totalSoles := float64(payment.AmountCents) / 100.0
	chargeID := fmt.Sprintf("yape-directo-%d", payment.ID)
	if err := s.Email.SendTicketEmail(
		payment.PassengerEmail, payment.PassengerName, reservationCode,
		nil, chargeID, totalSoles, payment.SeatLabels, payment.RouteName,
		saleID, s.Billing.TenantSlug,
	); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return s.ReservationRepo.MarkComprobanteEmailSent(ctx, payment.ID)
}

func (s *ComprobanteRetryService) createSale(ctx context.Context, payment *domain.Payment) (int, string) {
	if len(payment.SeatLabels) == 0 {
		return 0, "sin detalle de asientos"
	}
	idTdocumento := 3
	if payment.DocumentType == "factura" {
		idTdocumento = 1
	} else if payment.DocumentType == "pedido" {
		idTdocumento = 2
	}
	var detalle []BillingSaleDetail
	for _, label := range payment.SeatLabels {
		detalle = append(detalle, BillingSaleDetail{
			IDProducto:      1,
			ProdCodpro:      "777", // 777 = código genérico; el tenant usa detvDescripcion en vez de prodNombre
			IDUmedida:       1,
			DetvCantidad:    1,
			DetvPrecio:      payment.PricePerSeat,
			DetvDescripcion: fmt.Sprintf("ASIENTO %s - Pasaje %s", label, payment.RouteName),
		})
	}
	clienteID, err := s.Billing.FindOrCreateClient(ctx, payment.PassengerDocType, payment.PassengerDocNumber, payment.PassengerName, payment.PassengerAddress)
	if err != nil {
		return 0, fmt.Sprintf("cliente: %v", err)
	}
	sale := &BillingSaleRequest{
		VenSerie:         1,
		VenFecha:         time.Now().Format("2006-01-02"),
		VenCondiciones:   "Contado",
		Observaciones:    fmt.Sprintf("Pago Yape Directo / payment_id=%d / Doc: %s %s", payment.ID, payment.PassengerDocType, payment.PassengerDocNumber),
		DireccionFactura: payment.PassengerAddress,
		IDCliente:        clienteID,
		IDTdocumento:     idTdocumento,
		IDAlmacen:        1,
		IDTipoventa:      2,
		IDCorr:           1,
		Detalle:          detalle,
	}
	result, err := s.Billing.CreateSale(ctx, sale)
	if err != nil {
		if result != nil && result.SaleID > 0 {
			return result.SaleID, fmt.Sprintf("post-emisión: %v", err)
		}
		return 0, err.Error()
	}
	return result.SaleID, ""
}
