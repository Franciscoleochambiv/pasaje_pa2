package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/service"
	"pasaje/backend/internal/ws"

	"github.com/go-chi/chi/v5"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ParcelHandler maneja los endpoints de encomiendas.
type ParcelHandler struct {
	Parcel   *service.ParcelService
	Notifier *ws.Notifier
	Billing  *service.BillingService
	Routes   *service.RoutesService
}

// --- Público ---

// TrackParcel GET /api/parcels/track/{code}
func (h *ParcelHandler) TrackParcel(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "code es requerido")
		return
	}
	info, err := h.Parcel.ParcelRepo.GetPublicInfo(r.Context(), code)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Encomienda no encontrada")
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// --- Admin ---

// ListParcels GET /api/admin/parcels
func (h *ParcelHandler) ListParcels(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	var tripInstanceID int64
	if q := r.URL.Query().Get("trip_instance_id"); q != "" {
		tripInstanceID, _ = strconv.ParseInt(q, 10, 64)
	}
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	offset := 0
	if q := r.URL.Query().Get("offset"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 0 {
			offset = v
		}
	}

	list, total, err := h.Parcel.ParcelRepo.List(r.Context(), status, tripInstanceID, dateFrom, dateTo, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Parcel{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  list,
		"total": total,
	})
}

// GetParcel GET /api/admin/parcels/{id}
func (h *ParcelHandler) GetParcel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.Parcel.ParcelRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Encomienda no encontrada")
		return
	}
	tracking, err := h.Parcel.ParcelRepo.GetTracking(r.Context(), id)
	if err != nil {
		tracking = []domain.ParcelTracking{}
	}
	if tracking == nil {
		tracking = []domain.ParcelTracking{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"parcel":   p,
		"tracking": tracking,
	})
}

// CreateParcel POST /api/admin/parcels
func (h *ParcelHandler) CreateParcel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TripInstanceID     int64   `json:"trip_instance_id"`
		OriginStopID       int64   `json:"origin_stop_id"`
		DestStopID         int64   `json:"dest_stop_id"`
		SenderName         string  `json:"sender_name"`
		SenderDocType      string  `json:"sender_doc_type"`
		SenderDocNumber    string  `json:"sender_doc_number"`
		SenderPhone        string  `json:"sender_phone"`
		ReceiverName       string  `json:"receiver_name"`
		ReceiverDocType    string  `json:"receiver_doc_type"`
		ReceiverDocNumber  string  `json:"receiver_doc_number"`
		ReceiverPhone      string  `json:"receiver_phone"`
		PackageCount       int     `json:"package_count"`
		WeightKg           float64 `json:"weight_kg"`
		Description        string  `json:"description"`
		AmountCents        int64   `json:"amount_cents"`
		PaymentMode        string  `json:"payment_mode"`
		PaymentMethod      string  `json:"payment_method"`
		BillingDocType     string  `json:"billing_doc_type"`
		BillingEmail       string  `json:"billing_email"`
		BillingRuc         string  `json:"billing_ruc"`
		BillingRazonSocial string  `json:"billing_razon_social"`
		BillingAddress     string  `json:"billing_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	if body.TripInstanceID <= 0 || body.OriginStopID <= 0 || body.DestStopID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "trip_instance_id, origin_stop_id y dest_stop_id son requeridos")
		return
	}
	if body.OriginStopID == body.DestStopID {
		writeJSONError(w, http.StatusBadRequest, "origen y destino no pueden ser la misma parada")
		return
	}
	if body.SenderName == "" || body.SenderDocNumber == "" {
		writeJSONError(w, http.StatusBadRequest, "datos del remitente son requeridos")
		return
	}
	if body.ReceiverName == "" || body.ReceiverDocNumber == "" || body.ReceiverPhone == "" {
		writeJSONError(w, http.StatusBadRequest, "datos del destinatario son requeridos")
		return
	}
	if body.BillingEmail == "" {
		writeJSONError(w, http.StatusBadRequest, "email de facturación es requerido")
		return
	}
	if body.AmountCents <= 0 {
		writeJSONError(w, http.StatusBadRequest, "monto debe ser mayor a 0")
		return
	}

	if body.PaymentMode == "" {
		body.PaymentMode = "origin"
	}
	if body.PaymentMode != "origin" && body.PaymentMode != "destination" {
		writeJSONError(w, http.StatusBadRequest, "payment_mode debe ser 'origin' o 'destination'")
		return
	}
	if body.BillingDocType == "" {
		body.BillingDocType = "boleta"
	}
	if body.BillingDocType != "boleta" && body.BillingDocType != "factura" && body.BillingDocType != "pedido" {
		writeJSONError(w, http.StatusBadRequest, "billing_doc_type debe ser 'boleta', 'factura' o 'pedido'")
		return
	}
	if body.SenderDocType == "" {
		body.SenderDocType = "DNI"
	}
	if body.ReceiverDocType == "" {
		body.ReceiverDocType = "DNI"
	}
	if body.PackageCount <= 0 {
		body.PackageCount = 1
	}

	// Obtener user ID del contexto
	var registeredBy *int64
	if claims := UserFromContext(r.Context()); claims != nil {
		registeredBy = &claims.UserID
	}

	p := &domain.Parcel{
		TripInstanceID:     body.TripInstanceID,
		OriginStopID:       body.OriginStopID,
		DestStopID:         body.DestStopID,
		SenderName:         body.SenderName,
		SenderDocType:      body.SenderDocType,
		SenderDocNumber:    body.SenderDocNumber,
		SenderPhone:        body.SenderPhone,
		ReceiverName:       body.ReceiverName,
		ReceiverDocType:    body.ReceiverDocType,
		ReceiverDocNumber:  body.ReceiverDocNumber,
		ReceiverPhone:      body.ReceiverPhone,
		PackageCount:       body.PackageCount,
		WeightKg:           body.WeightKg,
		Description:        body.Description,
		AmountCents:        body.AmountCents,
		PaymentMode:        body.PaymentMode,
		PaymentMethod:      body.PaymentMethod,
		BillingDocType:     body.BillingDocType,
		BillingEmail:       body.BillingEmail,
		BillingRuc:         body.BillingRuc,
		BillingRazonSocial: body.BillingRazonSocial,
		BillingAddress:     body.BillingAddress,
		RegisteredBy:       registeredBy,
	}

	id, billingErrMsg, err := h.Parcel.Create(r.Context(), p)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	created, err := h.Parcel.ParcelRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "code": p.Code})
		return
	}

	if h.Notifier != nil {
		go h.Notifier.NotifyParcelUpdate(created.Code, "registered")
	}

	resp := map[string]any{"parcel": created}
	if billingErrMsg != "" {
		resp["billing_error"] = billingErrMsg
	}

	// Generar URLs de PDF y WhatsApp si hay billing_sale_id (boleta/factura o pedido)
	if created.BillingSaleID != nil && *created.BillingSaleID > 0 && h.Billing != nil {
		pdfURL := fmt.Sprintf("https://%s.facturame.online/api/factura/pdf/%d?format=ticket", h.Billing.TenantSlug, *created.BillingSaleID)
		resp["pdf_url"] = pdfURL
		phone := created.ReceiverPhone
		if created.PaymentMode == "origin" {
			phone = created.SenderPhone
		}
		if phone != "" {
			waMsg := fmt.Sprintf("Hola, su comprobante de encomienda %s esta disponible en: %s", created.Code, pdfURL)
			resp["whatsapp_url"] = fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(waMsg))
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

// UpdateParcel PUT /api/admin/parcels/{id}
func (h *ParcelHandler) UpdateParcel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	existing, err := h.Parcel.ParcelRepo.GetByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Encomienda no encontrada")
		return
	}
	if existing.Status == "delivered" || existing.Status == "cancelled" {
		writeJSONError(w, http.StatusConflict, "No se puede editar una encomienda entregada o anulada")
		return
	}

	var body struct {
		SenderName         string  `json:"sender_name"`
		SenderDocType      string  `json:"sender_doc_type"`
		SenderDocNumber    string  `json:"sender_doc_number"`
		SenderPhone        string  `json:"sender_phone"`
		ReceiverName       string  `json:"receiver_name"`
		ReceiverDocType    string  `json:"receiver_doc_type"`
		ReceiverDocNumber  string  `json:"receiver_doc_number"`
		ReceiverPhone      string  `json:"receiver_phone"`
		PackageCount       int     `json:"package_count"`
		WeightKg           float64 `json:"weight_kg"`
		Description        string  `json:"description"`
		AmountCents        int64   `json:"amount_cents"`
		PaymentMode        string  `json:"payment_mode"`
		BillingDocType     string  `json:"billing_doc_type"`
		BillingEmail       string  `json:"billing_email"`
		BillingRuc         string  `json:"billing_ruc"`
		BillingRazonSocial string  `json:"billing_razon_social"`
		BillingAddress     string  `json:"billing_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	p := &domain.Parcel{
		ID:                 id,
		SenderName:         body.SenderName,
		SenderDocType:      body.SenderDocType,
		SenderDocNumber:    body.SenderDocNumber,
		SenderPhone:        body.SenderPhone,
		ReceiverName:       body.ReceiverName,
		ReceiverDocType:    body.ReceiverDocType,
		ReceiverDocNumber:  body.ReceiverDocNumber,
		ReceiverPhone:      body.ReceiverPhone,
		PackageCount:       body.PackageCount,
		WeightKg:           body.WeightKg,
		Description:        body.Description,
		AmountCents:        body.AmountCents,
		PaymentMode:        body.PaymentMode,
		BillingDocType:     body.BillingDocType,
		BillingEmail:       body.BillingEmail,
		BillingRuc:         body.BillingRuc,
		BillingRazonSocial: body.BillingRazonSocial,
		BillingAddress:     body.BillingAddress,
	}

	if err := h.Parcel.ParcelRepo.Update(r.Context(), p); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// UpdateParcelStatus POST /api/admin/parcels/{id}/status
func (h *ParcelHandler) UpdateParcelStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body struct {
		Status   string `json:"status"`
		Location string `json:"location"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Status == "" {
		writeJSONError(w, http.StatusBadRequest, "status es requerido")
		return
	}

	var userID *int64
	if claims := UserFromContext(r.Context()); claims != nil {
		userID = &claims.UserID
	}

	if err := h.Parcel.UpdateStatus(r.Context(), id, body.Status, body.Location, body.Notes, userID); err != nil {
		msg := err.Error()
		code := http.StatusBadRequest
		if contains(msg, "no encontrada") {
			code = http.StatusNotFound
		} else if contains(msg, "no válida") {
			code = http.StatusUnprocessableEntity
		}
		writeJSONError(w, code, msg)
		return
	}

	// Broadcast WebSocket
	if h.Notifier != nil {
		p, _ := h.Parcel.ParcelRepo.GetByID(r.Context(), id)
		if p != nil {
			go h.Notifier.NotifyParcelUpdate(p.Code, body.Status)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// PayParcel POST /api/admin/parcels/{id}/pay
func (h *ParcelHandler) PayParcel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body struct {
		PaymentMethod string `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.PaymentMethod == "" {
		body.PaymentMethod = "efectivo"
	}

	updated, billingErrMsg, err := h.Parcel.Pay(r.Context(), id, body.PaymentMethod)
	if err != nil {
		msg := err.Error()
		code := http.StatusInternalServerError
		switch {
		case contains(msg, "no encontrada"):
			code = http.StatusNotFound
		case contains(msg, "ya está pagada"), contains(msg, "anulada"), contains(msg, "entregada"):
			code = http.StatusConflict
		}
		writeJSONError(w, code, msg)
		return
	}

	resp := map[string]any{
		"status": "paid",
		"parcel": updated,
	}
	if billingErrMsg != "" {
		resp["billing_error"] = billingErrMsg
	}

	if updated.BillingSaleID != nil && *updated.BillingSaleID > 0 && h.Billing != nil {
		pdfURL := fmt.Sprintf("https://%s.facturame.online/api/factura/pdf/%d?format=ticket", h.Billing.TenantSlug, *updated.BillingSaleID)
		resp["pdf_url"] = pdfURL
		phone := updated.ReceiverPhone
		if updated.PaymentMode == "origin" {
			phone = updated.SenderPhone
		}
		if phone != "" {
			waMsg := fmt.Sprintf("Hola, su comprobante de encomienda %s esta disponible en: %s", updated.Code, pdfURL)
			resp["whatsapp_url"] = fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(waMsg))
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// RetryParcelBilling POST /api/admin/parcels/{id}/retry-billing
// Re-intenta emitir el comprobante para una encomienda que quedó sin billing_sale_id.
func (h *ParcelHandler) RetryParcelBilling(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	updated, billingErrMsg, err := h.Parcel.RetryBilling(r.Context(), id)
	if err != nil {
		msg := err.Error()
		code := http.StatusInternalServerError
		if contains(msg, "no encontrada") {
			code = http.StatusNotFound
		} else if contains(msg, "anulada") {
			code = http.StatusConflict
		}
		writeJSONError(w, code, msg)
		return
	}

	resp := map[string]any{"parcel": updated}
	if billingErrMsg != "" {
		resp["billing_error"] = billingErrMsg
	}
	if updated.BillingSaleID != nil && *updated.BillingSaleID > 0 && h.Billing != nil {
		pdfURL := fmt.Sprintf("https://%s.facturame.online/api/factura/pdf/%d?format=ticket", h.Billing.TenantSlug, *updated.BillingSaleID)
		resp["pdf_url"] = pdfURL
		phone := updated.ReceiverPhone
		if updated.PaymentMode == "origin" {
			phone = updated.SenderPhone
		}
		if phone != "" {
			waMsg := fmt.Sprintf("Hola, su comprobante de encomienda %s esta disponible en: %s", updated.Code, pdfURL)
			resp["whatsapp_url"] = fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(waMsg))
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// CancelParcel POST /api/admin/parcels/{id}/cancel
func (h *ParcelHandler) CancelParcel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body struct {
		Notes string `json:"notes"`
	}
	// Notes es opcional, no fallar si body está vacío
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	var userID *int64
	if claims := UserFromContext(r.Context()); claims != nil {
		userID = &claims.UserID
	}
	traceID := fmt.Sprintf("void-parcel-%d-%d", id, time.Now().UnixNano())
	log.Printf("[void-trace:%s] parcel cancel requested id=%d by=%v", traceID, id, userID)
	writeVoidAudit(r.Context(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.requested", userID, map[string]any{
		"trace_id": traceID,
		"notes":    body.Notes,
	})

	cancelOut, err := h.Parcel.Cancel(r.Context(), id, body.Notes, userID)
	if err != nil {
		msg := err.Error()
		code := http.StatusBadRequest
		switch {
		case contains(msg, "no encontrada"):
			code = http.StatusNotFound
		case contains(msg, "entregada"), contains(msg, "anulada"):
			code = http.StatusConflict
		}
		writeJSONError(w, code, msg)
		return
	}
	log.Printf("[void-trace:%s] parcel cancel committed id=%d code=%s billing_sale_id=%v", traceID, id, cancelOut.Code, cancelOut.BillingSaleID)
	writeVoidAudit(r.Context(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.local_committed", userID, map[string]any{
		"trace_id":        traceID,
		"parcel_code":     cancelOut.Code,
		"billing_sale_id": cancelOut.BillingSaleID,
	})

	// Broadcast WebSocket
	if h.Notifier != nil && cancelOut != nil && cancelOut.Code != "" {
		go h.Notifier.NotifyParcelUpdate(cancelOut.Code, "cancelled")
	}

	// Cascadear anulación al tenant: DELETE /api/venta/{id} en Laravel + comunicación
	// de baja a SUNAT vía go-service. Igual patrón que yape_payment.VoidReservation.
	// Pedido (Tdoc=2) se omite del SUNAT por el go-service automáticamente (409).
	remoteVoidQueued := false
	if h.Billing != nil && cancelOut != nil && cancelOut.BillingSaleID != nil && *cancelOut.BillingSaleID > 0 {
		saleID := int(*cancelOut.BillingSaleID)
		remoteVoidQueued = true
		go func() {
			log.Printf("[void-trace:%s] parcel=%d sale=%d tenant void start", traceID, id, saleID)
			writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.tenant.start", userID, map[string]any{
				"trace_id": traceID,
				"sale_id":  saleID,
			})
			status, body, err := h.Billing.VoidSaleDetailed(context.Background(), saleID)
			if err != nil {
				log.Printf("[void-trace:%s] parcel=%d sale=%d tenant void error status=%d err=%v body=%q", traceID, id, saleID, status, err, truncateForLog(body))
				writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.tenant.error", userID, map[string]any{
					"trace_id": traceID,
					"sale_id":  saleID,
					"status":   status,
					"error":    err.Error(),
					"body":     truncateForLog(body),
				})
				return
			}
			log.Printf("[void-trace:%s] parcel=%d sale=%d tenant void ok status=%d body=%q", traceID, id, saleID, status, truncateForLog(body))
			writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.tenant.ok", userID, map[string]any{
				"trace_id": traceID,
				"sale_id":  saleID,
				"status":   status,
				"body":     truncateForLog(body),
			})
			log.Printf("[void-trace:%s] parcel=%d sale=%d sunat void start", traceID, id, saleID)
			writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.sunat.start", userID, map[string]any{
				"trace_id": traceID,
				"sale_id":  saleID,
			})
			sunatStatus, sunatBody, err := h.Billing.VoidSaleSunatDetailed(context.Background(), saleID)
			if err != nil {
				log.Printf("[void-trace:%s] parcel=%d sale=%d sunat void error status=%d err=%v body=%q", traceID, id, saleID, sunatStatus, err, truncateForLog(sunatBody))
				writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.sunat.error", userID, map[string]any{
					"trace_id": traceID,
					"sale_id":  saleID,
					"status":   sunatStatus,
					"error":    err.Error(),
					"body":     truncateForLog(sunatBody),
				})
			} else {
				log.Printf("[void-trace:%s] parcel=%d sale=%d sunat void ok status=%d body=%q", traceID, id, saleID, sunatStatus, truncateForLog(sunatBody))
				writeVoidAudit(context.Background(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.sunat.ok", userID, map[string]any{
					"trace_id": traceID,
					"sale_id":  saleID,
					"status":   sunatStatus,
					"body":     truncateForLog(sunatBody),
				})
			}
		}()
	} else if h.Billing == nil {
		log.Printf("[void-trace:%s] parcel=%d billing service disabled; remote void skipped", traceID, id)
		writeVoidAudit(r.Context(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.remote.skipped", userID, map[string]any{
			"trace_id": traceID,
			"reason":   "billing service disabled",
		})
	} else {
		log.Printf("[void-trace:%s] parcel=%d remote void not queued: billing_sale_id empty", traceID, id)
		writeVoidAudit(r.Context(), h.Parcel.ParcelRepo.Pool, "parcel", &id, "void.remote.skipped", userID, map[string]any{
			"trace_id": traceID,
			"reason":   "billing_sale_id empty",
		})
	}

	resp := map[string]any{"status": "cancelled"}
	if cancelOut != nil && cancelOut.BillingSaleID != nil {
		resp["billing_sale_id"] = *cancelOut.BillingSaleID
	}
	resp["void_trace_id"] = traceID
	resp["remote_void_queued"] = remoteVoidQueued
	if !remoteVoidQueued {
		resp["remote_void_reason"] = "billing_sale_id ausente o no configurado"
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListParcelsByTrip GET /api/admin/parcels/by-trip/{tripId}
func (h *ParcelHandler) ListParcelsByTrip(w http.ResponseWriter, r *http.Request) {
	tripID, err := strconv.ParseInt(chi.URLParam(r, "tripId"), 10, 64)
	if err != nil || tripID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "tripId inválido")
		return
	}
	list, err := h.Parcel.ParcelRepo.ListByTrip(r.Context(), tripID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Parcel{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GetTripManifest GET /api/admin/trips/{id}/manifest
func (h *ParcelHandler) GetTripManifest(w http.ResponseWriter, r *http.Request) {
	tripID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || tripID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	// Get trip seat info (includes passengers via sold seats)
	seatResp, err := h.Routes.GetTripSeatResponse(r.Context(), tripID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("error cargando viaje: %v", err))
		return
	}

	// Get parcels for this trip
	parcelList, err := h.Parcel.ParcelRepo.ListByTrip(r.Context(), tripID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("error cargando encomiendas: %v", err))
		return
	}
	if parcelList == nil {
		parcelList = []domain.Parcel{}
	}

	// Count passengers (sold seats)
	passengerCount := 0
	for _, s := range seatResp.Seats {
		if s.Status == "sold" {
			passengerCount++
		}
	}

	// Calculate parcel revenue
	var parcelRevenue int64
	for _, p := range parcelList {
		if p.Status != "cancelled" {
			parcelRevenue += p.AmountCents
		}
	}

	activeParcels := 0
	for _, p := range parcelList {
		if p.Status != "cancelled" {
			activeParcels++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"trip": map[string]any{
			"id":             tripID,
			"route_name":     seatResp.RouteName,
			"vehicle_name":   seatResp.VehicleName,
			"price_per_seat": seatResp.PricePerSeat,
		},
		"seats":   seatResp.Seats,
		"parcels": parcelList,
		"totals": map[string]any{
			"passengers":         passengerCount,
			"parcels":            activeParcels,
			"revenue_passengers": float64(passengerCount) * seatResp.PricePerSeat,
			"revenue_parcels":    float64(parcelRevenue) / 100.0,
		},
	})
}
