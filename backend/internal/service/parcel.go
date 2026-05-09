package service

import (
	"context"
	"fmt"
	"time"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
)

// ParcelService lógica de negocio de encomiendas.
type ParcelService struct {
	ParcelRepo *repository.ParcelRepository
	Billing    *BillingService
	Email      *EmailService
}

// Create registra una nueva encomienda. Si pago en origen y método dado, cobra inmediatamente.
// Si pago en destino, genera un pedido (documento tipo 2) en el sistema de ventas.
func (s *ParcelService) Create(ctx context.Context, p *domain.Parcel) (int64, string, error) {
	p.Code = repository.GenerateParcelCode()
	p.Currency = "PEN"
	p.Status = "registered"

	if p.PaymentMode == "origin" && p.PaymentMethod != "" {
		p.PaymentStatus = "paid"
	} else if p.PaymentMode == "destination" {
		p.PaymentStatus = "pending"
	} else {
		p.PaymentStatus = "pending"
	}

	id, err := s.ParcelRepo.Create(ctx, p)
	if err != nil {
		return 0, "", err
	}
	p.ID = id

	if full, err := s.ParcelRepo.GetByID(ctx, id); err == nil {
		p.OriginStopName = full.OriginStopName
		p.DestStopName = full.DestStopName
		p.RouteName = full.RouteName
	}

	billingErrMsg := ""
	if p.PaymentStatus == "paid" {
		saleID, billingErr := s.emitBilling(ctx, p)
		if billingErr != nil {
			billingErrMsg = billingErr.Error()
			fmt.Printf("parcel billing: error emitting for %s: %v\n", p.Code, billingErr)
		} else if saleID > 0 {
			if err := s.ParcelRepo.SetBillingSaleID(ctx, id, saleID); err != nil {
				billingErrMsg = err.Error()
				fmt.Printf("parcel billing: error saving sale id for %s: %v\n", p.Code, err)
			} else {
				s.sendParcelEmail(p, saleID)
			}
		}
	} else if p.PaymentMode == "destination" {
		saleID, billingErr := s.emitPedido(ctx, p)
		if billingErr != nil {
			billingErrMsg = billingErr.Error()
			fmt.Printf("parcel pedido: error emitting for %s: %v\n", p.Code, billingErr)
		} else if saleID > 0 {
			_ = s.ParcelRepo.SetBillingSaleID(ctx, id, saleID)
		}
	}

	return id, billingErrMsg, nil
}

// UpdateStatus cambia el estado de una encomienda validando transiciones.
func (s *ParcelService) UpdateStatus(ctx context.Context, id int64, newStatus, location, notes string, userID *int64) error {
	p, err := s.ParcelRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("encomienda no encontrada: %w", err)
	}

	valid := domain.ValidParcelTransitions[p.Status]
	allowed := false
	for _, v := range valid {
		if v == newStatus {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("transición no válida de '%s' a '%s'", p.Status, newStatus)
	}

	return s.ParcelRepo.UpdateStatus(ctx, id, newStatus, location, notes, userID)
}

// Pay cobra una encomienda pendiente y emite comprobante.
// El segundo retorno es un mensaje de error de billing no-fatal: el cobro local
// se persiste igual, y el admin puede reintentar la emisión más tarde.
func (s *ParcelService) Pay(ctx context.Context, id int64, paymentMethod string) (*domain.Parcel, string, error) {
	p, err := s.ParcelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("encomienda no encontrada: %w", err)
	}

	if p.PaymentStatus == "paid" {
		return nil, "", fmt.Errorf("la encomienda ya está pagada")
	}
	if p.Status == "cancelled" {
		return nil, "", fmt.Errorf("no se puede cobrar una encomienda anulada")
	}
	if p.Status == "delivered" {
		return nil, "", fmt.Errorf("no se puede cobrar una encomienda ya entregada")
	}

	// Emitir comprobante. Si falla, no abortamos el cobro: marcamos paid sin
	// billing_sale_id y dejamos que el admin reintente vía /retry-billing.
	saleID, billingErr := s.emitBilling(ctx, p)
	billingErrMsg := ""
	if billingErr != nil {
		billingErrMsg = billingErr.Error()
		fmt.Printf("parcel pay: error emitiendo comprobante para %s: %v\n", p.Code, billingErr)
	}

	var saleIDPtr *int64
	if saleID > 0 {
		saleIDPtr = &saleID
	}

	if err := s.ParcelRepo.MarkPaid(ctx, id, paymentMethod, saleIDPtr); err != nil {
		return nil, billingErrMsg, err
	}

	if saleID > 0 {
		s.sendParcelEmail(p, saleID)
	}

	updated, err := s.ParcelRepo.GetByID(ctx, id)
	return updated, billingErrMsg, err
}

// RetryBilling re-intenta la emisión del comprobante para una encomienda que
// quedó pagada pero sin billing_sale_id (fallo transitorio del tenant/go-service).
func (s *ParcelService) RetryBilling(ctx context.Context, id int64) (*domain.Parcel, string, error) {
	p, err := s.ParcelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("encomienda no encontrada: %w", err)
	}
	if p.BillingSaleID != nil && *p.BillingSaleID > 0 {
		return p, "", nil // ya emitido
	}
	if p.Status == "cancelled" {
		return nil, "", fmt.Errorf("no se puede emitir comprobante para encomienda anulada")
	}

	var saleID int64
	var billingErr error
	if p.PaymentMode == "destination" && p.PaymentStatus != "paid" {
		saleID, billingErr = s.emitPedido(ctx, p)
	} else {
		saleID, billingErr = s.emitBilling(ctx, p)
	}
	if billingErr != nil {
		return p, billingErr.Error(), nil
	}
	if saleID <= 0 {
		return p, "no se generó comprobante", nil
	}
	if err := s.ParcelRepo.SetBillingSaleID(ctx, id, saleID); err != nil {
		return nil, "", err
	}
	s.sendParcelEmail(p, saleID)
	updated, err := s.ParcelRepo.GetByID(ctx, id)
	return updated, "", err
}

// Cancel anula una encomienda.
func (s *ParcelService) Cancel(ctx context.Context, id int64, notes string, userID *int64) (*repository.CancelParcelResult, error) {
	p, err := s.ParcelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("encomienda no encontrada: %w", err)
	}

	if p.Status == "delivered" {
		return nil, fmt.Errorf("no se puede anular una encomienda entregada")
	}
	if p.Status == "cancelled" {
		return nil, fmt.Errorf("la encomienda ya está anulada")
	}

	return s.ParcelRepo.Cancel(ctx, id, notes, userID)
}

// emitBilling crea la boleta/factura/pedido en el sistema Venta.
func (s *ParcelService) emitBilling(ctx context.Context, p *domain.Parcel) (int64, error) {
	if s.Billing == nil {
		return 0, nil
	}

	// ID_Tdocumento: 1=Factura, 2=Pedido, 3=Boleta
	idTdocumento := 3
	if p.BillingDocType == "factura" {
		idTdocumento = 1
	} else if p.BillingDocType == "pedido" {
		idTdocumento = 2
	}

	// Determinar datos del cliente para facturación
	docType := p.SenderDocType
	docNumber := p.SenderDocNumber
	clientName := p.SenderName
	address := p.BillingAddress

	if p.BillingDocType == "factura" && p.BillingRuc != "" {
		docType = "RUC"
		docNumber = p.BillingRuc
		clientName = p.BillingRazonSocial
	}

	clienteID, err := s.Billing.FindOrCreateClient(ctx, docType, docNumber, clientName, address)
	if err != nil {
		return 0, fmt.Errorf("error buscando/creando cliente: %w", err)
	}

	priceFloat := float64(p.AmountCents) / 100.0

	// Descripción con código de encomienda y detalle para el sistema de ventas
	desc := fmt.Sprintf("%s - %s→%s - %d bultos",
		p.Code, p.OriginStopName, p.DestStopName, p.PackageCount)
	if p.Description != "" {
		desc += " | " + p.Description
	}
	if p.WeightKg > 0 {
		desc += fmt.Sprintf(" (%.1fkg)", p.WeightKg)
	}

	sale := &BillingSaleRequest{
		VenSerie:         1,
		VenFecha:         time.Now().Format("2006-01-02"),
		VenCondiciones:   "Contado",
		VenImporteLetras: AmountInLettersPEN(priceFloat),
		VenNroguia:       "",
		VenSaldo:         0,
		Observaciones:    fmt.Sprintf("Encomienda: %s / %s %s / %s→%s", p.Code, docType, docNumber, p.OriginStopName, p.DestStopName),
		DireccionFactura: address,
		IDCliente:        clienteID,
		IDEmpleado:       nil,
		IDTdocumento:     idTdocumento,
		IDAlmacen:        1,
		IDTipoventa:      2,
		IDCorr:           1,
		IDVendedor:       nil,
		Detalle: []BillingSaleDetail{
			{
				IDProducto:      1,
				ProdCodpro:      "777",
				IDUmedida:       1,
				DetvCantidad:    1,
				DetvPrecio:      priceFloat,
				DetvDescripcion: desc,
			},
		},
	}

	result, err := s.Billing.CreateSale(ctx, sale)
	if err != nil {
		return 0, err
	}

	return int64(result.SaleID), nil
}

// emitPedido genera un pedido (documento tipo 2) para encomiendas con pago en destino.
func (s *ParcelService) emitPedido(ctx context.Context, p *domain.Parcel) (int64, error) {
	if s.Billing == nil {
		return 0, nil
	}

	docType := p.SenderDocType
	docNumber := p.SenderDocNumber
	clientName := p.SenderName
	address := p.BillingAddress

	clienteID, err := s.Billing.FindOrCreateClient(ctx, docType, docNumber, clientName, address)
	if err != nil {
		return 0, fmt.Errorf("error buscando/creando cliente: %w", err)
	}

	priceFloat := float64(p.AmountCents) / 100.0

	desc := fmt.Sprintf("%s - %s→%s - %d bultos",
		p.Code, p.OriginStopName, p.DestStopName, p.PackageCount)
	if p.Description != "" {
		desc += " | " + p.Description
	}
	if p.WeightKg > 0 {
		desc += fmt.Sprintf(" (%.1fkg)", p.WeightKg)
	}

	sale := &BillingSaleRequest{
		VenSerie:         1,
		VenFecha:         time.Now().Format("2006-01-02"),
		VenCondiciones:   "Pago en destino",
		VenImporteLetras: AmountInLettersPEN(priceFloat),
		VenNroguia:       "",
		VenSaldo:         priceFloat,
		Observaciones:    fmt.Sprintf("PEDIDO Encomienda: %s / %s %s / %s→%s / Destinatario: %s %s", p.Code, docType, docNumber, p.OriginStopName, p.DestStopName, p.ReceiverName, p.ReceiverPhone),
		DireccionFactura: address,
		IDCliente:        clienteID,
		IDEmpleado:       nil,
		IDTdocumento:     2, // Pedido
		IDAlmacen:        1,
		IDTipoventa:      2,
		IDCorr:           1,
		IDVendedor:       nil,
		Detalle: []BillingSaleDetail{
			{
				IDProducto:      1,
				ProdCodpro:      "777",
				IDUmedida:       1,
				DetvCantidad:    1,
				DetvPrecio:      priceFloat,
				DetvDescripcion: desc,
			},
		},
	}

	result, err := s.Billing.CreateSale(ctx, sale)
	if err != nil {
		return 0, err
	}

	return int64(result.SaleID), nil
}

// sendParcelEmail envía email con comprobante electrónico.
func (s *ParcelService) sendParcelEmail(p *domain.Parcel, billingSaleID int64) {
	if s.Email == nil || p.BillingEmail == "" {
		return
	}

	go func() {
		totalSoles := float64(p.AmountCents) / 100.0
		err := s.Email.SendParcelEmail(
			p.BillingEmail,
			p.SenderName,
			p.Code,
			p.OriginStopName,
			p.DestStopName,
			p.ReceiverName,
			p.PackageCount,
			p.WeightKg,
			totalSoles,
			int(billingSaleID),
			s.Billing.TenantSlug,
		)
		if err != nil {
			fmt.Printf("email: error sending parcel receipt to %s: %v\n", p.BillingEmail, err)
		} else {
			fmt.Printf("email: sent parcel receipt to %s for %s\n", p.BillingEmail, p.Code)
		}
	}()
}
