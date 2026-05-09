package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

// BillingHandler maneja los endpoints /api/billing/*.
type BillingHandler struct {
	Billing         *service.BillingService
	Email           *service.EmailService
	ReservationRepo *repository.ReservationRepository
}

// LookupDNI GET /api/billing/dni/{dni}
func (h *BillingHandler) LookupDNI(w http.ResponseWriter, r *http.Request) {
	dni := chi.URLParam(r, "dni")
	if len(dni) != 8 {
		writeJSONError(w, http.StatusBadRequest, "DNI debe tener 8 digitos")
		return
	}
	result, err := h.Billing.LookupDNI(r.Context(), dni)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// LookupRUC GET /api/billing/ruc/{ruc}
func (h *BillingHandler) LookupRUC(w http.ResponseWriter, r *http.Request) {
	ruc := chi.URLParam(r, "ruc")
	if len(ruc) != 11 {
		writeJSONError(w, http.StatusBadRequest, "RUC debe tener 11 digitos")
		return
	}
	result, err := h.Billing.LookupRUC(r.Context(), ruc)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// CreateBillingSale POST /api/billing/sale
func (h *BillingHandler) CreateBillingSale(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DocumentType       string   `json:"document_type"` // boleta | factura | pedido
		TripInstanceID     int      `json:"trip_instance_id"`
		ReservationCode    string   `json:"reservation_code"`
		PassengerName      string   `json:"passenger_name"`
		PassengerDocType   string   `json:"passenger_doc_type"`
		PassengerDocNumber string   `json:"passenger_doc_number"`
		PassengerAddress   string   `json:"passenger_address"`
		PricePerSeat       float64  `json:"price_per_seat"`
		RouteName          string   `json:"route_name"`
		SeatLabels         []string `json:"seat_labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	if body.DocumentType == "" {
		body.DocumentType = "boleta"
	}

	// Determine document type ID: 1=Factura, 2=Pedido, 3=Boleta
	idTdocumento := 3
	if body.DocumentType == "factura" {
		idTdocumento = 1
	} else if body.DocumentType == "pedido" {
		idTdocumento = 2
	}

	// Build detail lines - one per seat
	var detalle []service.BillingSaleDetail
	for _, label := range body.SeatLabels {
		detalle = append(detalle, service.BillingSaleDetail{
			IDProducto:      1,
			ProdCodpro:      "777", // 777 = código genérico; el tenant usa detvDescripcion en vez de prodNombre
			IDUmedida:       1,
			DetvCantidad:    1,
			DetvPrecio:      body.PricePerSeat,
			DetvDescripcion: fmt.Sprintf("ASIENTO %s - Pasaje %s", label, body.RouteName),
		})
	}

	if len(detalle) == 0 {
		writeJSONError(w, http.StatusBadRequest, "seat_labels es requerido")
		return
	}

	// Buscar o crear cliente en el tenant de facturación
	clienteID, err := h.Billing.FindOrCreateClient(r.Context(), body.PassengerDocType, body.PassengerDocNumber, body.PassengerName, body.PassengerAddress)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("Error buscando/creando cliente: %v", err))
		return
	}

	totalAmount := body.PricePerSeat * float64(len(body.SeatLabels))
	sale := &service.BillingSaleRequest{
		VenSerie:         1, // el Go Service de venta resuelve la serie y correlativo
		VenFecha:         time.Now().Format("2006-01-02"),
		VenCondiciones:   "Contado",
		VenImporteLetras: service.AmountInLettersPEN(totalAmount),
		VenNroguia:       "",
		VenSaldo:         0,
		Observaciones:    fmt.Sprintf("Reserva: %s / Doc: %s %s", body.ReservationCode, body.PassengerDocType, body.PassengerDocNumber),
		DireccionFactura: body.PassengerAddress,
		IDCliente:        clienteID,
		IDEmpleado:       nil,
		IDTdocumento:     idTdocumento,
		IDAlmacen:        1,
		IDTipoventa:      2,
		IDCorr:           1, // correlativo default, triggers en BD manejan la numeracion
		IDVendedor:       nil,
		Detalle:          detalle,
	}

	result, err := h.Billing.CreateSale(r.Context(), sale)
	if err != nil {
		if result != nil && result.SaleID > 0 {
			if h.ReservationRepo != nil && body.ReservationCode != "" {
				if saveErr := h.ReservationRepo.SetReservationBillingSaleIDByCode(r.Context(), body.ReservationCode, result.SaleID); saveErr != nil {
					writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("comprobante emitido pero no se pudo guardar billing_sale_id: %v", saveErr))
					return
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"billing_sale_id": result.SaleID,
				"status":          "partial_success",
				"message":         fmt.Sprintf("comprobante emitido con observaciones: %v", err),
				"warnings":        result.Warnings,
			})
			return
		}

		writeJSON(w, http.StatusBadGateway, map[string]any{
			"billing_sale_id": 0,
			"status":          "error",
			"message":         err.Error(),
		})
		return
	}

	if h.ReservationRepo != nil && body.ReservationCode != "" {
		if err := h.ReservationRepo.SetReservationBillingSaleIDByCode(r.Context(), body.ReservationCode, result.SaleID); err != nil {
			writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("comprobante emitido pero no se pudo guardar billing_sale_id: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"billing_sale_id": result.SaleID,
		"status":          result.Status,
		"message":         result.Message,
		"warnings":        result.Warnings,
	})
}

// GetPDF GET /api/billing/pdf/{ventaId}
func (h *BillingHandler) GetPDF(w http.ResponseWriter, r *http.Request) {
	ventaIDStr := chi.URLParam(r, "ventaId")
	ventaID, err := strconv.Atoi(ventaIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "ventaId invalido")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "ticket"
	}

	pdfBytes, err := h.Billing.GetPDF(r.Context(), ventaID, format)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=boleta-%d.pdf", ventaID))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

// SendReceiptEmail POST /api/billing/send-email — send receipt email with PDF/XML/CDR links
func (h *BillingHandler) SendReceiptEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email           string   `json:"email"`
		PassengerName   string   `json:"passenger_name"`
		ReservationCode string   `json:"reservation_code"`
		TicketCodes     []string `json:"ticket_codes"`
		TotalAmount     float64  `json:"total_amount"`
		SeatLabels      []string `json:"seat_labels"`
		RouteName       string   `json:"route_name"`
		BillingSaleID   int      `json:"billing_sale_id"`
		PaymentRef      string   `json:"payment_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	if body.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "Email es requerido")
		return
	}

	if h.Email == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Servicio de email no configurado")
		return
	}

	// Send async
	go func() {
		err := h.Email.SendTicketEmail(
			body.Email,
			body.PassengerName,
			body.ReservationCode,
			body.TicketCodes,
			body.PaymentRef,
			body.TotalAmount,
			body.SeatLabels,
			body.RouteName,
			body.BillingSaleID,
			h.Billing.TenantSlug,
		)
		if err != nil {
			fmt.Printf("email: error sending to %s: %v\n", body.Email, err)
		} else {
			fmt.Printf("email: sent receipt to %s (venta=%d)\n", body.Email, body.BillingSaleID)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Correo enviandose a %s", body.Email),
	})
}
