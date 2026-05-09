package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailService sends emails via SMTP (Gmail).
type EmailService struct {
	Host     string
	Port     string
	Username string
	Password string
	FromAddr string
	FromName string
}

// SendTicketEmail sends the ticket/receipt info to the passenger.
// billingSaleID > 0 means we have a comprobante and can include PDF/XML/CDR links.
func (s *EmailService) SendTicketEmail(to, passengerName, reservationCode string, ticketCodes []string, chargeID string, totalAmount float64, seatLabels []string, routeName string, billingSaleID int, tenantSlug string) error {
	if s.Host == "" || s.Username == "" {
		return fmt.Errorf("email not configured")
	}

	seats := strings.Join(seatLabels, ", ")
	tickets := strings.Join(ticketCodes, ", ")

	subject := fmt.Sprintf("Tu pasaje %s - Codigo: %s", routeName, reservationCode)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:'Segoe UI',Arial,sans-serif;background:#f8fafc;margin:0;padding:0">
<div style="max-width:600px;margin:0 auto;background:white;border-radius:12px;overflow:hidden;box-shadow:0 2px 12px rgba(0,0,0,0.08);margin-top:20px;margin-bottom:20px">

  <!-- Header -->
  <div style="background:linear-gradient(135deg,#0f1d4e,#1b55f5);padding:30px;text-align:center">
    <h1 style="color:white;margin:0;font-size:24px;font-weight:800">Pasaje Confirmado</h1>
    <p style="color:rgba(255,255,255,0.8);margin:8px 0 0;font-size:14px">Tu compra fue procesada exitosamente</p>
  </div>

  <!-- Content -->
  <div style="padding:30px">
    <p style="color:#334155;font-size:15px;line-height:1.6">
      Hola <strong>%s</strong>,<br>
      Tu pasaje ha sido confirmado. Aqui estan los detalles de tu viaje:
    </p>

    <!-- Details Card -->
    <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:20px;margin:20px 0">
      <table style="width:100%%;border-collapse:collapse">
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600">Codigo de Reserva</td>
          <td style="padding:8px 0;text-align:right;font-weight:800;color:#1b55f5;font-size:16px;font-family:monospace;letter-spacing:1px">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Ruta</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Asientos</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Total Pagado</td>
          <td style="padding:8px 0;text-align:right;font-weight:800;color:#059669;font-size:18px;border-top:1px solid #e2e8f0">S/ %.2f</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Boletos</td>
          <td style="padding:8px 0;text-align:right;font-weight:600;color:#1e293b;font-family:monospace;font-size:12px;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Cargo</td>
          <td style="padding:8px 0;text-align:right;font-weight:500;color:#94a3b8;font-size:11px;border-top:1px solid #e2e8f0">%s</td>
        </tr>
      </table>
    </div>

    <!-- Comprobante Downloads -->
    %s

    <!-- Instructions -->
    <div style="background:#ecfdf5;border:1px solid #a7f3d0;border-radius:10px;padding:15px;margin:20px 0">
      <p style="color:#065f46;font-size:13px;margin:0;line-height:1.5">
        <strong>Instrucciones:</strong><br>
        Presenta este correo o tu codigo de reserva <strong>%s</strong> al momento de abordar el bus.
      </p>
    </div>

    <p style="color:#94a3b8;font-size:12px;text-align:center;margin-top:25px">
      Gracias por viajar con nosotros.<br>
      Pasaje — Sistema de Reserva de Pasajes Interprovinciales
    </p>
  </div>
</div>
</body>
</html>`,
		passengerName,
		reservationCode,
		routeName,
		seats,
		totalAmount,
		tickets,
		chargeID,
		comprobanteSection(billingSaleID, tenantSlug),
		reservationCode,
	)

	return s.sendHTML(to, subject, body)
}

func comprobanteSection(billingSaleID int, tenantSlug string) string {
	if billingSaleID <= 0 || tenantSlug == "" {
		return ""
	}
	baseURL := fmt.Sprintf("https://%s.facturame.online", tenantSlug)
	pdfURL := fmt.Sprintf("%s/api/factura/pdf/%d?format=ticket", baseURL, billingSaleID)
	xmlURL := fmt.Sprintf("%s/api/venta/%d/xml-public", baseURL, billingSaleID)
	cdrURL := fmt.Sprintf("%s/api/venta/%d/cdr-public", baseURL, billingSaleID)

	return fmt.Sprintf(`
    <div style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:10px;padding:20px;margin:20px 0;text-align:center">
      <p style="color:#1e40af;font-size:14px;font-weight:700;margin:0 0 12px">Comprobante Electronico</p>
      <table style="width:100%%;border-collapse:collapse">
        <tr>
          <td style="padding:6px">
            <a href="%s" style="display:inline-block;padding:10px 20px;background:#1b55f5;color:white;border-radius:8px;text-decoration:none;font-size:13px;font-weight:700">
              PDF Boleta/Factura
            </a>
          </td>
          <td style="padding:6px">
            <a href="%s" style="display:inline-block;padding:10px 20px;background:#059669;color:white;border-radius:8px;text-decoration:none;font-size:13px;font-weight:700">
              XML Comprobante
            </a>
          </td>
          <td style="padding:6px">
            <a href="%s" style="display:inline-block;padding:10px 20px;background:#7c3aed;color:white;border-radius:8px;text-decoration:none;font-size:13px;font-weight:700">
              CDR SUNAT
            </a>
          </td>
        </tr>
      </table>
    </div>`, pdfURL, xmlURL, cdrURL)
}

// SendParcelEmail sends the parcel receipt to the sender.
func (s *EmailService) SendParcelEmail(to, senderName, parcelCode, originStop, destStop, receiverName string, packageCount int, weightKg, totalAmount float64, billingSaleID int, tenantSlug string) error {
	if s.Host == "" || s.Username == "" {
		return fmt.Errorf("email not configured")
	}

	subject := fmt.Sprintf("Encomienda %s - %s → %s", parcelCode, originStop, destStop)

	weightStr := ""
	if weightKg > 0 {
		weightStr = fmt.Sprintf("%.1f kg", weightKg)
	} else {
		weightStr = "-"
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:'Segoe UI',Arial,sans-serif;background:#f8fafc;margin:0;padding:0">
<div style="max-width:600px;margin:0 auto;background:white;border-radius:12px;overflow:hidden;box-shadow:0 2px 12px rgba(0,0,0,0.08);margin-top:20px;margin-bottom:20px">

  <div style="background:linear-gradient(135deg,#0f1d4e,#7c3aed);padding:30px;text-align:center">
    <h1 style="color:white;margin:0;font-size:24px;font-weight:800">Encomienda Registrada</h1>
    <p style="color:rgba(255,255,255,0.8);margin:8px 0 0;font-size:14px">Comprobante de envío</p>
  </div>

  <div style="padding:30px">
    <p style="color:#334155;font-size:15px;line-height:1.6">
      Hola <strong>%s</strong>,<br>
      Tu encomienda ha sido registrada exitosamente.
    </p>

    <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:20px;margin:20px 0">
      <table style="width:100%%;border-collapse:collapse">
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600">Codigo</td>
          <td style="padding:8px 0;text-align:right;font-weight:800;color:#7c3aed;font-size:16px;font-family:monospace;letter-spacing:1px">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Origen</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Destino</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Destinatario</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Bultos</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%d</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Peso</td>
          <td style="padding:8px 0;text-align:right;font-weight:700;color:#1e293b;border-top:1px solid #e2e8f0">%s</td>
        </tr>
        <tr>
          <td style="padding:8px 0;color:#64748b;font-size:13px;font-weight:600;border-top:1px solid #e2e8f0">Total</td>
          <td style="padding:8px 0;text-align:right;font-weight:800;color:#059669;font-size:18px;border-top:1px solid #e2e8f0">S/ %.2f</td>
        </tr>
      </table>
    </div>

    %s

    <div style="background:#ecfdf5;border:1px solid #a7f3d0;border-radius:10px;padding:15px;margin:20px 0">
      <p style="color:#065f46;font-size:13px;margin:0;line-height:1.5">
        <strong>Seguimiento:</strong><br>
        Puedes rastrear tu encomienda con el codigo <strong>%s</strong> en nuestra pagina web.
      </p>
    </div>

    <p style="color:#94a3b8;font-size:12px;text-align:center;margin-top:25px">
      Gracias por confiar en nosotros.<br>
      Pasaje — Sistema de Encomiendas
    </p>
  </div>
</div>
</body>
</html>`,
		senderName, parcelCode, originStop, destStop, receiverName,
		packageCount, weightStr, totalAmount,
		comprobanteSection(billingSaleID, tenantSlug),
		parcelCode,
	)

	return s.sendHTML(to, subject, body)
}

func (s *EmailService) sendHTML(to, subject, htmlBody string) error {
	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", s.FromName, s.FromAddr),
		"To":           to,
		"Subject":      subject,
		"MIME-Version":  "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	tlsConfig := &tls.Config{
		ServerName: s.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(s.FromAddr); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	return client.Quit()
}
