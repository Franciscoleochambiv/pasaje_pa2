package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AmountInLettersPEN convierte un monto en soles a su representacion en letras
// con el formato usado en comprobantes peruanos: "SON: VEINTE SOLES CON 50/100".
// Devuelve solo la parte de "VEINTE SOLES CON 50/100" (sin el "SON:").
func AmountInLettersPEN(amount float64) string {
	cents := int(math.Round(amount*100)) % 100
	whole := int(math.Floor(amount + 1e-9))
	wordsRaw := strings.TrimSpace(numberToWordsES(whole))
	if wordsRaw == "" {
		wordsRaw = "CERO"
	}
	soles := "SOLES"
	if whole == 1 {
		soles = "SOL"
	}
	return strings.ToUpper(fmt.Sprintf("%s %s CON %02d/100", wordsRaw, soles, cents))
}

func numberToWordsES(n int) string {
	if n < 0 {
		return "MENOS " + numberToWordsES(-n)
	}
	if n == 0 {
		return "cero"
	}

	units := []string{"", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve",
		"diez", "once", "doce", "trece", "catorce", "quince", "dieciseis", "diecisiete", "dieciocho", "diecinueve",
		"veinte", "veintiuno", "veintidos", "veintitres", "veinticuatro", "veinticinco", "veintiseis", "veintisiete", "veintiocho", "veintinueve"}
	tens := []string{"", "", "", "treinta", "cuarenta", "cincuenta", "sesenta", "setenta", "ochenta", "noventa"}
	hundreds := []string{"", "ciento", "doscientos", "trescientos", "cuatrocientos", "quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos"}

	switch {
	case n < 30:
		return units[n]
	case n < 100:
		t := n / 10
		u := n % 10
		if u == 0 {
			return tens[t]
		}
		return tens[t] + " y " + units[u]
	case n == 100:
		return "cien"
	case n < 1000:
		h := n / 100
		rest := n % 100
		if rest == 0 {
			return hundreds[h]
		}
		return hundreds[h] + " " + numberToWordsES(rest)
	case n < 1_000_000:
		thousands := n / 1000
		rest := n % 1000
		var thousandsPart string
		if thousands == 1 {
			thousandsPart = "mil"
		} else {
			thousandsPart = numberToWordsES(thousands) + " mil"
		}
		if rest == 0 {
			return thousandsPart
		}
		return thousandsPart + " " + numberToWordsES(rest)
	default:
		millions := n / 1_000_000
		rest := n % 1_000_000
		var millionsPart string
		if millions == 1 {
			millionsPart = "un millon"
		} else {
			millionsPart = numberToWordsES(millions) + " millones"
		}
		if rest == 0 {
			return millionsPart
		}
		return millionsPart + " " + numberToWordsES(rest)
	}
}

// BillingService connects to the Venta Go Service for billing integration.
type BillingService struct {
	GoServiceURL     string
	TenantSlug       string
	APIPeruURL       string
	APIPeruToken     string
	TenantDBPool     *pgxpool.Pool // direct connection to tenant DB (local dev)
	TenantAPIURL     string        // Laravel tenant API URL (e.g., https://ancalla.facturame.online)
	TenantEmail      string        // login credentials for tenant API
	TenantPassword   string
	TenantToken      string // cached Sanctum token (exported for config handler)
	TenantTokenExpAt time.Time
}

// DNIResult holds the response from API Peru DNI lookup.
type DNIResult struct {
	Nombres         string `json:"nombres"`
	ApellidoPaterno string `json:"apellido_paterno"`
	ApellidoMaterno string `json:"apellido_materno"`
	NombreCompleto  string `json:"nombre_completo"`
	DNI             string `json:"dni"`
}

// RUCResult holds the response from API Peru RUC lookup.
type RUCResult struct {
	RUC          string `json:"ruc"`
	RazonSocial  string `json:"razon_social"`
	NombreORazon string `json:"nombre_o_razon_social"`
	Estado       string `json:"estado"`
	Condicion    string `json:"condicion"`
	Direccion    string `json:"direccion"`
}

// BillingSaleRequest is the payload sent to the Go Service to create a sale.
type BillingSaleRequest struct {
	VenSerie         int                 `json:"venSerie"`
	VenFecha         string              `json:"venFecha"`
	VenCondiciones   string              `json:"venCondiciones"`
	VenImporteLetras string              `json:"venImporteLetras"`
	VenNroguia       string              `json:"venNroguia"`
	VenSaldo         float64             `json:"venSaldo"`
	Observaciones    string              `json:"observaciones"`
	DireccionFactura string              `json:"direccion_factura,omitempty"`
	IDCliente        int                 `json:"ID_Cliente"`
	IDEmpleado       *int                `json:"ID_Empleado"`
	IDTdocumento     int                 `json:"ID_Tdocumento"` // 1=Factura, 2=Pedido, 3=Boleta
	IDAlmacen        int                 `json:"ID_Almacen"`
	IDTipoventa      int                 `json:"ID_Tipoventa"`
	IDCorr           int                 `json:"ID_Corr"`
	IDVendedor       *int                `json:"ID_Vendedor"`
	Detalle          []BillingSaleDetail `json:"detalle"`
}

// BillingSaleDetail is a line item in a billing sale.
type BillingSaleDetail struct {
	IDProducto      int     `json:"ID_Producto"`
	ProdCodpro      string  `json:"prodCodpro"`
	IDUmedida       int     `json:"ID_Umedida"`
	DetvCantidad    float64 `json:"detvCantidad"`
	DetvPrecio      float64 `json:"detvPrecio"`
	DetvDescripcion string  `json:"detvDescripcion"`
}

// BillingSaleResponse is the response from the Go Service after creating a sale.
type BillingSaleResponse struct {
	Status   string   `json:"status"`
	Message  string   `json:"message"`
	SaleID   int      `json:"id_venta"`
	Warnings []string `json:"warnings"`
}

// apiPeruResponse wraps the API Peru response format: { success, message, data }
type apiPeruResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// LookupDNI searches for a person by DNI using API Peru.
func (s *BillingService) LookupDNI(ctx context.Context, dni string) (*DNIResult, error) {
	body, _ := json.Marshal(map[string]string{"dni": dni})
	req, err := http.NewRequestWithContext(ctx, "POST", s.APIPeruURL+"/api/dni", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.APIPeruToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIPeruToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api peru: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api peru: status %d", resp.StatusCode)
	}

	var wrapper apiPeruResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}
	if !wrapper.Success || wrapper.Data == nil {
		return nil, fmt.Errorf("DNI no encontrado")
	}

	var result DNIResult
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LookupRUC searches for a company by RUC using API Peru.
func (s *BillingService) LookupRUC(ctx context.Context, ruc string) (*RUCResult, error) {
	body, _ := json.Marshal(map[string]string{"ruc": ruc})
	req, err := http.NewRequestWithContext(ctx, "POST", s.APIPeruURL+"/api/ruc", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.APIPeruToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIPeruToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api peru: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("api peru: status %d", resp.StatusCode)
	}

	var wrapper apiPeruResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}
	if !wrapper.Success || wrapper.Data == nil {
		return nil, fmt.Errorf("RUC no encontrado")
	}

	var result RUCResult
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateSale sends a sale to the Venta Go Service for billing.
func (s *BillingService) CreateSale(ctx context.Context, sale *BillingSaleRequest) (*BillingSaleResponse, error) {
	body, err := json.Marshal(sale)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.GoServiceURL+"/api/sales", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant", s.TenantSlug)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("go service: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result BillingSaleResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("go service response: %s", string(respBody))
	}

	if result.Status != "success" {
		return &result, fmt.Errorf("go service: %s", result.Message)
	}

	return &result, nil
}

// ResetToken forces re-authentication on next request.
func (s *BillingService) ResetToken() {
	s.TenantToken = ""
	s.TenantTokenExpAt = time.Time{}
}

// ensureTenantToken autentica con la API Laravel del tenant si el token expiró.
func (s *BillingService) ensureTenantToken(ctx context.Context) error {
	if s.TenantToken != "" && time.Now().Before(s.TenantTokenExpAt) {
		return nil // token vigente
	}

	loginURL := s.TenantAPIURL + "/api/login"
	body, _ := json.Marshal(map[string]string{
		"email":    s.TenantEmail,
		"password": s.TenantPassword,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("login tenant: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login tenant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login tenant: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("login tenant parse: %w", err)
	}

	s.TenantToken = result.Token
	s.TenantTokenExpAt = time.Now().Add(23 * time.Hour) // token dura 24h, renovamos a las 23h
	return nil
}

// tenantAPIRequest hace una petición autenticada a la API Laravel del tenant.
// Si recibe 401, refresca el token y reintenta una vez.
func (s *BillingService) TenantAPIRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	doRequest := func() ([]byte, int, error) {
		if err := s.ensureTenantToken(ctx); err != nil {
			return nil, 0, err
		}

		var bodyReader io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		}

		req, err := http.NewRequestWithContext(ctx, method, s.TenantAPIURL+path, bodyReader)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.TenantToken)

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		return respBody, resp.StatusCode, nil
	}

	respBody, status, err := doRequest()
	if err != nil {
		return nil, 0, err
	}

	// Si 401, refrescar token y reintentar
	if status == 401 {
		s.ResetToken()
		return doRequest()
	}

	return respBody, status, nil
}

// VoidSaleDetailed anula una venta en el tenant Laravel via DELETE /api/venta/{id}
// y devuelve status/body para trazabilidad.
func (s *BillingService) VoidSaleDetailed(ctx context.Context, saleID int) (int, string, error) {
	if saleID <= 0 || s.TenantAPIURL == "" {
		return 0, "", nil
	}
	respBody, status, err := s.TenantAPIRequest(ctx, "DELETE", fmt.Sprintf("/api/venta/%d", saleID), nil)
	if err != nil {
		return 0, "", fmt.Errorf("tenant DELETE venta %d: %w", saleID, err)
	}
	body := string(respBody)
	if status >= 200 && status < 300 {
		return status, body, nil
	}
	return status, body, fmt.Errorf("tenant DELETE venta %d: status %d body=%s", saleID, status, body)
}

// VoidSale anula una venta en el tenant Laravel via DELETE /api/venta/{id}.
// Devuelve nil si responde 2xx; cualquier otro caso retorna error sin bloquear
// la anulación local previa.
func (s *BillingService) VoidSale(ctx context.Context, saleID int) error {
	_, _, err := s.VoidSaleDetailed(ctx, saleID)
	return err
}

// VoidSaleSunatDetailed dispara la comunicación de baja a SUNAT y devuelve
// status/body para trazabilidad.
func (s *BillingService) VoidSaleSunatDetailed(ctx context.Context, saleID int) (int, string, error) {
	if saleID <= 0 || s.GoServiceURL == "" {
		return 0, "", nil
	}
	body, _ := json.Marshal(map[string]int{"venta_id": saleID})
	req, err := http.NewRequestWithContext(ctx, "POST", s.GoServiceURL+"/api/venta/anular-sunat", bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant", s.TenantSlug)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("sunat anular venta %d: %w", saleID, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	bodyText := string(respBody)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, bodyText, nil
	}
	if resp.StatusCode == http.StatusConflict {
		// pedido u otro tipo no-CPE: no es error
		return resp.StatusCode, bodyText, nil
	}
	return resp.StatusCode, bodyText, fmt.Errorf("sunat anular venta %d: status %d body=%s", saleID, resp.StatusCode, bodyText)
}

// VoidSaleSunat dispara la comunicación de baja a SUNAT via go-service:
// POST {GoServiceURL}/api/venta/anular-sunat con {"venta_id": id}.
// El go-service se encarga de generar RA (factura) o RC (boleta) y mandar a backenweb.
// Para pedidos responde 409 (no aplica), que tratamos como no-error.
func (s *BillingService) VoidSaleSunat(ctx context.Context, saleID int) error {
	_, _, err := s.VoidSaleSunatDetailed(ctx, saleID)
	return err
}

// FindOrCreateClient busca un cliente por numero de documento en el tenant.
// Primero intenta via API Laravel. Si no hay API, usa conexión directa a PostgreSQL.
func (s *BillingService) FindOrCreateClient(ctx context.Context, docType, docNumber, name, address string) (int, error) {
	// Mapear tipo de documento: DNI=1, RUC=6, Pasaporte=7, CE=4
	cliTipodoc := "1"
	switch docType {
	case "RUC":
		cliTipodoc = "6"
	case "Pasaporte":
		cliTipodoc = "7"
	case "CE":
		cliTipodoc = "4"
	}

	// Opción 1: Usar API Laravel del tenant (producción - K8s)
	if s.TenantAPIURL != "" && s.TenantEmail != "" {
		return s.findOrCreateClientViaAPI(ctx, cliTipodoc, docNumber, name, address)
	}

	// Opción 2: Conexión directa a PostgreSQL del tenant (desarrollo local)
	if s.TenantDBPool != nil {
		return s.findOrCreateClientViaDB(ctx, cliTipodoc, docNumber, name, address)
	}

	return 1, nil // fallback: cliente default
}

func (s *BillingService) findOrCreateClientViaAPI(ctx context.Context, cliTipodoc, docNumber, name, address string) (int, error) {
	// Buscar cliente existente por documento
	respBody, status, err := s.TenantAPIRequest(ctx, "GET", "/api/cliente?q="+docNumber, nil)
	if err != nil {
		return 0, fmt.Errorf("buscando cliente via API: %w", err)
	}

	// Si 401, forzar re-login y reintentar
	if status == 401 {
		s.ResetToken()
		respBody, status, err = s.TenantAPIRequest(ctx, "GET", "/api/cliente?q="+docNumber, nil)
		if err != nil {
			return 0, fmt.Errorf("buscando cliente via API (retry): %w", err)
		}
	}

	if status == 200 {
		// Parse response - could be paginated { data: [...] } or direct array
		var searchResult struct {
			Data []struct {
				IDCliente int    `json:"ID_Cliente"`
				CliNdoc   string `json:"cliNdoc"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &searchResult); err == nil {
			for _, c := range searchResult.Data {
				if c.CliNdoc == docNumber {
					return c.IDCliente, nil
				}
			}
		}
	}

	// No encontrado, crear
	createBody := map[string]interface{}{
		"cliNombre":       name,
		"cliApellidos":    "",
		"cliTelefono":     "",
		"cliCorreo":       "",
		"cliNdoc":         docNumber,
		"cliTipodoc":      cliTipodoc,
		"cliDireccion":    address,
		"cliLineacredito": 0,
	}

	respBody, status, err = s.TenantAPIRequest(ctx, "POST", "/api/cliente", createBody)
	if err != nil {
		return 0, fmt.Errorf("creando cliente via API: %w", err)
	}

	if status == 201 || status == 200 {
		var created struct {
			IDCliente int `json:"ID_Cliente"`
		}
		if err := json.Unmarshal(respBody, &created); err == nil && created.IDCliente > 0 {
			return created.IDCliente, nil
		}
	}

	// Si falla por duplicado (ya existe), buscar de nuevo
	if status == 422 {
		respBody, _, err = s.TenantAPIRequest(ctx, "GET", "/api/cliente?q="+docNumber, nil)
		if err == nil {
			var retry struct {
				Data []struct {
					IDCliente int    `json:"ID_Cliente"`
					CliNdoc   string `json:"cliNdoc"`
				} `json:"data"`
			}
			if json.Unmarshal(respBody, &retry) == nil {
				for _, c := range retry.Data {
					if c.CliNdoc == docNumber {
						return c.IDCliente, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("no se pudo crear cliente: status %d, response: %s", status, string(respBody))
}

func (s *BillingService) findOrCreateClientViaDB(ctx context.Context, cliTipodoc, docNumber, name, address string) (int, error) {
	var clienteID int
	err := s.TenantDBPool.QueryRow(ctx, `
		SELECT "ID_Cliente" FROM clientes WHERE "cliNdoc" = $1 LIMIT 1
	`, docNumber).Scan(&clienteID)

	if err == nil {
		return clienteID, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("buscando cliente: %w", err)
	}

	err = s.TenantDBPool.QueryRow(ctx, `
		INSERT INTO clientes ("cliNombre", "cliApellidos", "cliTelefono", "cliCorreo", "cliNdoc", "cliTipodoc", "cliDireccion", "cliLineacredito", "cliFecha", "created_at", "updated_at")
		VALUES ($1, '', '', '', $2, $3, $4, 0, NOW(), NOW(), NOW())
		RETURNING "ID_Cliente"
	`, name, docNumber, cliTipodoc, address).Scan(&clienteID)

	if err != nil {
		return 0, fmt.Errorf("creando cliente: %w", err)
	}
	return clienteID, nil
}

// GetPDF retrieves the PDF for a sale from the Venta Laravel backend.
func (s *BillingService) GetPDF(ctx context.Context, ventaID int, format string) ([]byte, error) {
	url := fmt.Sprintf("https://%s.facturame.online/api/factura/pdf/%d?format=%s", s.TenantSlug, ventaID, format)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/pdf")
	req.Header.Set("X-Tenant", s.TenantSlug)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pdf service: status %d: %s", resp.StatusCode, string(b))
	}

	return io.ReadAll(resp.Body)
}
