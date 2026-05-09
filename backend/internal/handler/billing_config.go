package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
)

// BillingConfigHandler manages billing configuration via admin panel.
type BillingConfigHandler struct {
	Billing  *service.BillingService
	Settings *repository.SettingsRepository
}

type billingConfigDTO struct {
	GoServiceURL   string `json:"go_service_url"`
	TenantSlug     string `json:"tenant_slug"`
	APIPeruURL     string `json:"apiperu_url"`
	APIPeruToken   string `json:"apiperu_token"`
	TenantEmail    string `json:"tenant_email"`
	TenantPassword string `json:"tenant_password"`
}

// GetConfig GET /api/admin/billing-config
func (h *BillingConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	pass := ""
	if h.Billing.TenantPassword != "" {
		pass = "••••••••"
	}
	writeJSON(w, http.StatusOK, billingConfigDTO{
		GoServiceURL:   h.Billing.GoServiceURL,
		TenantSlug:     h.Billing.TenantSlug,
		APIPeruURL:     h.Billing.APIPeruURL,
		APIPeruToken:   h.Billing.APIPeruToken,
		TenantEmail:    h.Billing.TenantEmail,
		TenantPassword: pass,
	})
}

// UpdateConfig PUT /api/admin/billing-config
func (h *BillingConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var body billingConfigDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	if body.GoServiceURL != "" {
		h.Billing.GoServiceURL = body.GoServiceURL
	}
	if body.TenantSlug != "" {
		h.Billing.TenantSlug = body.TenantSlug
		h.Billing.TenantAPIURL = fmt.Sprintf("https://%s.facturame.online", body.TenantSlug)
	}
	if body.APIPeruURL != "" {
		h.Billing.APIPeruURL = body.APIPeruURL
	}
	if body.APIPeruToken != "" {
		h.Billing.APIPeruToken = body.APIPeruToken
	}
	if body.TenantEmail != "" {
		h.Billing.TenantEmail = body.TenantEmail
	}
	if body.TenantPassword != "" && body.TenantPassword != "••••••••" {
		h.Billing.TenantPassword = body.TenantPassword
		h.Billing.ResetToken()
	}

	// Auto-sync empresa data after changing tenant
	if body.TenantSlug != "" {
		go h.syncEmpresaData()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Configuracion actualizada"})
}

// ListTenants GET /api/admin/billing-config/tenants — list available tenants from venta landlord
func (h *BillingConfigHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	// Login to landlord
	loginBody, _ := json.Marshal(map[string]string{
		"email":    h.Billing.TenantEmail,
		"password": h.Billing.TenantPassword,
	})

	client := &http.Client{Timeout: 15 * time.Second}
	loginResp, err := client.Post("https://adminfactura.facturame.online/api/landlord/login",
		"application/json", bytes.NewReader(loginBody))
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Error conectando al landlord: "+err.Error())
		return
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != 200 {
		b, _ := io.ReadAll(loginResp.Body)
		writeJSONError(w, http.StatusBadGateway, "Login landlord fallo: "+string(b))
		return
	}

	var loginResult struct {
		Token string `json:"token"`
	}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)

	// Get tenants
	req, _ := http.NewRequest("GET", "https://adminfactura.facturame.online/api/landlord/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+loginResult.Token)
	req.Header.Set("Accept", "application/json")

	tenantsResp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Error obteniendo tenants: "+err.Error())
		return
	}
	defer tenantsResp.Body.Close()

	body, _ := io.ReadAll(tenantsResp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// TestConnection GET /api/admin/billing-config/test
func (h *BillingConfigHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	results := make(map[string]interface{})
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Go Service health
	goOK := false
	resp, err := client.Get(h.Billing.GoServiceURL + "/health")
	if err != nil {
		results["go_service"] = fmt.Sprintf("Error: %v", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 200 {
			goOK = true
			results["go_service"] = "Conectado"
		} else {
			results["go_service"] = fmt.Sprintf("Status %d: %s", resp.StatusCode, string(body))
		}
	}

	// 2. Tenant login
	tenantOK := false
	if h.Billing.TenantAPIURL != "" && h.Billing.TenantEmail != "" {
		loginBody, _ := json.Marshal(map[string]string{
			"email":    h.Billing.TenantEmail,
			"password": h.Billing.TenantPassword,
		})
		loginURL := h.Billing.TenantAPIURL + "/api/login"
		req, _ := http.NewRequest("POST", loginURL, bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		resp2, err := client.Do(req)
		if err != nil {
			results["tenant_login"] = fmt.Sprintf("Error: %v", err)
		} else {
			defer resp2.Body.Close()
			body, _ := io.ReadAll(resp2.Body)
			if resp2.StatusCode == 200 {
				tenantOK = true
				results["tenant_login"] = "Autenticado correctamente"
			} else {
				results["tenant_login"] = fmt.Sprintf("Status %d: %s", resp2.StatusCode, string(body))
			}
		}
	} else {
		results["tenant_login"] = "No configurado"
	}

	// 3. API Peru
	apiPeruOK := false
	if h.Billing.APIPeruURL != "" {
		testBody, _ := json.Marshal(map[string]string{"dni": "10000001"})
		req, _ := http.NewRequest("POST", h.Billing.APIPeruURL+"/api/dni", bytes.NewReader(testBody))
		req.Header.Set("Content-Type", "application/json")
		if h.Billing.APIPeruToken != "" {
			req.Header.Set("Authorization", "Bearer "+h.Billing.APIPeruToken)
		}
		resp3, err := client.Do(req)
		if err != nil {
			results["api_peru"] = fmt.Sprintf("Error: %v", err)
		} else {
			defer resp3.Body.Close()
			if resp3.StatusCode == 200 {
				apiPeruOK = true
				results["api_peru"] = "Conectado"
			} else {
				results["api_peru"] = fmt.Sprintf("Status %d", resp3.StatusCode)
			}
		}
	} else {
		results["api_peru"] = "No configurado"
	}

	status := "error"
	msg := "Algunos servicios no estan disponibles"
	if goOK && tenantOK && apiPeruOK {
		status = "ok"
		msg = fmt.Sprintf("Go Service OK · Tenant %s OK · API Peru OK", h.Billing.TenantSlug)
	} else if goOK && tenantOK {
		status = "ok"
		msg = fmt.Sprintf("Go Service OK · Tenant %s OK · API Peru: %s", h.Billing.TenantSlug, results["api_peru"])
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  status,
		"message": msg,
		"details": results,
	})
}

// SubscriptionInfo describes our facturación tenant subscription as exposed by the landlord.
type SubscriptionInfo struct {
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	IsActive        bool    `json:"is_active"`
	TrialEndsAt     *string `json:"trial_ends_at"`
	SubscribedUntil *string `json:"subscribed_until"`
	DaysRemaining   *int    `json:"days_remaining"`
	Status          string  `json:"status"` // active | expiring_soon | expired | trial | unknown
}

// GetSubscription GET /api/admin/billing-config/subscription
// Reads our tenant entry from the landlord and reports subscription validity so the
// admin dashboard can warn before the billing API stops working.
func (h *BillingConfigHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	if h.Billing == nil || h.Billing.TenantSlug == "" {
		writeJSONError(w, http.StatusBadRequest, "Tenant de facturación no configurado")
		return
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    h.Billing.TenantEmail,
		"password": h.Billing.TenantPassword,
	})
	client := &http.Client{Timeout: 12 * time.Second}
	loginResp, err := client.Post("https://adminfactura.facturame.online/api/landlord/login",
		"application/json", bytes.NewReader(loginBody))
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "landlord login: "+err.Error())
		return
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != 200 {
		b, _ := io.ReadAll(loginResp.Body)
		writeJSONError(w, http.StatusBadGateway, "landlord login fallo: "+string(b))
		return
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil || login.Token == "" {
		writeJSONError(w, http.StatusBadGateway, "respuesta landlord invalida")
		return
	}

	req, _ := http.NewRequest("GET", "https://adminfactura.facturame.online/api/landlord/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	req.Header.Set("Accept", "application/json")
	tenantsResp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "tenants: "+err.Error())
		return
	}
	defer tenantsResp.Body.Close()

	var tenants []struct {
		Name            string  `json:"name"`
		Slug            string  `json:"slug"`
		IsActive        bool    `json:"is_active"`
		TrialEndsAt     *string `json:"trial_ends_at"`
		SubscribedUntil *string `json:"subscribed_until"`
	}
	if err := json.NewDecoder(tenantsResp.Body).Decode(&tenants); err != nil {
		writeJSONError(w, http.StatusBadGateway, "tenants payload invalido: "+err.Error())
		return
	}

	info := SubscriptionInfo{Slug: h.Billing.TenantSlug, Status: "unknown"}
	for _, t := range tenants {
		if t.Slug == h.Billing.TenantSlug {
			info.Name = t.Name
			info.IsActive = t.IsActive
			info.TrialEndsAt = t.TrialEndsAt
			info.SubscribedUntil = t.SubscribedUntil
			break
		}
	}
	end := info.SubscribedUntil
	if end == nil || *end == "" {
		end = info.TrialEndsAt
	}
	if end != nil && *end != "" {
		if ts, err := time.Parse(time.RFC3339, *end); err == nil {
			days := int(time.Until(ts).Hours() / 24)
			info.DaysRemaining = &days
			switch {
			case !info.IsActive:
				info.Status = "expired"
			case days < 0:
				info.Status = "expired"
			case days <= 7:
				info.Status = "expiring_soon"
			case info.SubscribedUntil != nil && *info.SubscribedUntil != "":
				info.Status = "active"
			default:
				info.Status = "trial"
			}
		}
	} else if !info.IsActive {
		info.Status = "expired"
	}

	writeJSON(w, http.StatusOK, info)
}

// SyncEmpresa POST /api/admin/billing-config/sync-empresa — sync empresa data to settings
func (h *BillingConfigHandler) SyncEmpresa(w http.ResponseWriter, r *http.Request) {
	updates, err := h.doSyncEmpresa()
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": "Datos sincronizados", "data": updates})
}

func (h *BillingConfigHandler) syncEmpresaData() {
	h.doSyncEmpresa()
}

func (h *BillingConfigHandler) doSyncEmpresa() (map[string]string, error) {
	if h.Settings == nil || h.Billing == nil {
		return nil, fmt.Errorf("settings or billing not configured")
	}

	respBody, status, err := h.Billing.TenantAPIRequest(context.Background(), "GET", "/api/empresat", nil)
	if err != nil {
		return nil, fmt.Errorf("conectando con venta: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("venta respondio con status %d", status)
	}

	var empresaResp struct {
		Success bool `json:"success"`
		Data    struct {
			RazonEmisor   string  `json:"razon_emisor"`
			RucEmisor     string  `json:"ruc_emisor"`
			Direccion     string  `json:"direccion"`
			Provincia     string  `json:"provincia"`
			Ciudad        string  `json:"ciudad"`
			Distrito      string  `json:"distrito"`
			Email         *string `json:"email"`
			Telefonos     string  `json:"telefonos"`
			Moneda        string  `json:"moneda"`
			PorcentajeIGV float64 `json:"porcentaje_igv"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &empresaResp); err != nil || !empresaResp.Success {
		return nil, fmt.Errorf("datos de empresa no disponibles")
	}

	d := empresaResp.Data
	dir := d.Direccion
	if d.Distrito != "" {
		dir += ", " + d.Distrito
	}
	if d.Provincia != "" {
		dir += ", " + d.Provincia
	}
	if d.Ciudad != "" {
		dir += " - " + d.Ciudad
	}

	email := ""
	if d.Email != nil {
		email = *d.Email
	}

	updates := map[string]string{
		"company_name":    d.RazonEmisor,
		"company_ruc":     d.RucEmisor,
		"company_address": dir,
		"company_phone":   d.Telefonos,
		"company_email":   email,
		"currency":        d.Moneda,
	}

	ctx := context.Background()
	h.Settings.SetBulk(ctx, updates)
	return updates, nil
}
