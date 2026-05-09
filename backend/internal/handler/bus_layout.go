package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pasaje/backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

// ListBusLayouts GET /api/admin/bus-layouts
func (h *AdminHandler) ListBusLayouts(w http.ResponseWriter, r *http.Request) {
	list, err := h.Admin.ListBusLayouts(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.BusLayout{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GetBusLayoutFull GET /api/admin/bus-layouts/{id}/full
func (h *AdminHandler) GetBusLayoutFull(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	full, err := h.Admin.GetBusLayoutFull(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "plantilla no encontrada")
		return
	}
	writeJSON(w, http.StatusOK, full)
}

// CreateBusLayout POST /api/admin/bus-layouts
func (h *AdminHandler) CreateBusLayout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name            string `json:"name"`
		Brand           string `json:"brand"`
		Model           string `json:"model"`
		Description     string `json:"description"`
		Floors          int    `json:"floors"`
		LayoutCols      int    `json:"layout_cols"`
		SeatType        string `json:"seat_type"`
		PreviewImageURL string `json:"preview_image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name es requerido")
		return
	}
	if body.Floors <= 0 {
		body.Floors = 1
	}
	if body.LayoutCols <= 0 {
		body.LayoutCols = 4
	}
	if body.SeatType == "" {
		body.SeatType = "regular"
	}
	layout := &domain.BusLayout{
		Name: body.Name, Brand: body.Brand, Model: body.Model, Description: body.Description,
		Floors: body.Floors, LayoutCols: body.LayoutCols, SeatType: body.SeatType,
		PreviewImageURL: body.PreviewImageURL,
	}
	id, err := h.Admin.CreateBusLayout(r.Context(), layout)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateBusLayout PUT /api/admin/bus-layouts/{id}
func (h *AdminHandler) UpdateBusLayout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name            string `json:"name"`
		Brand           string `json:"brand"`
		Model           string `json:"model"`
		Description     string `json:"description"`
		Floors          int    `json:"floors"`
		LayoutCols      int    `json:"layout_cols"`
		SeatType        string `json:"seat_type"`
		PreviewImageURL string `json:"preview_image_url"`
		Active          *bool  `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	layout := &domain.BusLayout{
		ID: id, Name: body.Name, Brand: body.Brand, Model: body.Model, Description: body.Description,
		Floors: body.Floors, LayoutCols: body.LayoutCols, SeatType: body.SeatType,
		PreviewImageURL: body.PreviewImageURL, Active: active,
	}
	if err := h.Admin.UpdateBusLayout(r.Context(), layout); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteBusLayout DELETE /api/admin/bus-layouts/{id}
func (h *AdminHandler) DeleteBusLayout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.Admin.DeleteBusLayout(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SaveBusLayoutFull PUT /api/admin/bus-layouts/{id}/full
// Reemplaza atómicamente metadata + asientos + elementos.
func (h *AdminHandler) SaveBusLayoutFull(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Layout struct {
			Name            string `json:"name"`
			Brand           string `json:"brand"`
			Model           string `json:"model"`
			Description     string `json:"description"`
			Floors          int    `json:"floors"`
			LayoutCols      int    `json:"layout_cols"`
			SeatType        string `json:"seat_type"`
			PreviewImageURL string `json:"preview_image_url"`
			Active          *bool  `json:"active"`
		} `json:"layout"`
		Seats []struct {
			Label    string `json:"label"`
			Position int    `json:"position"`
			Floor    int    `json:"floor"`
			RowNum   int    `json:"row_num"`
			ColNum   int    `json:"col_num"`
			SeatType string `json:"seat_type"`
		} `json:"seats"`
		Elements []struct {
			Floor  int    `json:"floor"`
			RowNum int    `json:"row_num"`
			ColNum int    `json:"col_num"`
			Kind   string `json:"kind"`
			Text   string `json:"text"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	active := true
	if body.Layout.Active != nil {
		active = *body.Layout.Active
	}
	layout := &domain.BusLayout{
		ID: id, Name: body.Layout.Name, Brand: body.Layout.Brand, Model: body.Layout.Model,
		Description: body.Layout.Description, Floors: body.Layout.Floors, LayoutCols: body.Layout.LayoutCols,
		SeatType: body.Layout.SeatType, PreviewImageURL: body.Layout.PreviewImageURL, Active: active,
	}
	seats := make([]domain.BusLayoutSeat, 0, len(body.Seats))
	for _, s := range body.Seats {
		st := s.SeatType
		if st == "" {
			st = "regular"
		}
		seats = append(seats, domain.BusLayoutSeat{
			LayoutID: id, Label: s.Label, Position: s.Position,
			Floor: s.Floor, RowNum: s.RowNum, ColNum: s.ColNum, SeatType: st,
		})
	}
	elements := make([]domain.BusLayoutElement, 0, len(body.Elements))
	for _, e := range body.Elements {
		elements = append(elements, domain.BusLayoutElement{
			LayoutID: id, Floor: e.Floor, RowNum: e.RowNum, ColNum: e.ColNum, Kind: e.Kind, Text: e.Text,
		})
	}
	if err := h.Admin.SaveBusLayoutFull(r.Context(), layout, seats, elements); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// SaveVehicleAsLayout POST /api/admin/vehicles/{id}/save-as-layout
func (h *AdminHandler) SaveVehicleAsLayout(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || vehicleID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name            string `json:"name"`
		Brand           string `json:"brand"`
		Model           string `json:"model"`
		Description     string `json:"description"`
		PreviewImageURL string `json:"preview_image_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name es requerido")
		return
	}
	id, err := h.Admin.SaveVehicleAsLayout(r.Context(), vehicleID, body.Name, body.Brand, body.Model, body.Description, body.PreviewImageURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
