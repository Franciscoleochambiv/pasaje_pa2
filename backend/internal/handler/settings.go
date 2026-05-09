package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"
)

type SettingsHandler struct {
	Repo    *repository.SettingsRepository
	Billing *service.BillingService
}

// GetAll GET /api/admin/settings — returns all settings
func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.GetAll(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []repository.Setting{}
	}
	writeJSON(w, http.StatusOK, list)
}

// Update PUT /api/admin/settings — bulk update settings
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON invalido")
		return
	}
	body := make(map[string]string, len(raw))
	for k, v := range raw {
		body[k] = fmt.Sprintf("%v", v)
	}
	if err := h.Repo.SetBulk(r.Context(), body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SyncFromVenta POST /api/admin/settings/sync — pulls empresa data from Venta tenant
func (h *SettingsHandler) SyncFromVenta(w http.ResponseWriter, r *http.Request) {
	if h.Billing == nil {
		writeJSONError(w, http.StatusBadRequest, "Billing no configurado")
		return
	}

	// Call tenant API to get empresa data
	respBody, status, err := h.Billing.TenantAPIRequest(r.Context(), "GET", "/api/empresat", nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Error conectando con venta: "+err.Error())
		return
	}
	if status != 200 {
		writeJSONError(w, http.StatusBadGateway, "Venta respondio con status "+string(rune(status+'0')))
		return
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

	if err := json.Unmarshal(respBody, &empresaResp); err != nil {
		writeJSONError(w, http.StatusBadGateway, "Error parseando respuesta de venta")
		return
	}

	if !empresaResp.Success {
		writeJSONError(w, http.StatusBadGateway, "Venta no devolvio datos de empresa")
		return
	}

	d := empresaResp.Data
	dirFull := d.Direccion
	if d.Distrito != "" {
		dirFull += ", " + d.Distrito
	}
	if d.Provincia != "" {
		dirFull += ", " + d.Provincia
	}
	if d.Ciudad != "" {
		dirFull += " - " + d.Ciudad
	}

	email := ""
	if d.Email != nil {
		email = *d.Email
	}

	updates := map[string]string{
		"company_name":    d.RazonEmisor,
		"company_ruc":     d.RucEmisor,
		"company_address": dirFull,
		"company_phone":   d.Telefonos,
		"company_email":   email,
		"currency":        d.Moneda,
	}

	if err := h.Repo.SetBulk(r.Context(), updates); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "Datos sincronizados desde " + d.RazonEmisor,
		"data":    updates,
	})
}

// GetPublic GET /api/settings/public — returns non-sensitive settings for frontend
func (h *SettingsHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.GetAll(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Only expose safe keys
	safe := map[string]string{}
	safeKeys := map[string]bool{
		"currency": true, "company_name": true,
		"company_phone": true, "company_email": true, "company_address": true,
		"igv_percent": true, "whatsapp_phone": true, "yape_business_number": true,
	}
	for _, s := range list {
		if safeKeys[s.Key] {
			safe[s.Key] = s.Value
		}
	}
	writeJSON(w, http.StatusOK, safe)
}
