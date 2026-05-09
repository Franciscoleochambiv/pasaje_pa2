package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/repository"
	"pasaje/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

// UserHandler maneja los endpoints /api/admin/users.
type UserHandler struct {
	UserRepo *repository.UserRepository
}

// ListUsers GET /api/admin/users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.UserRepo.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// CreateUser POST /api/admin/users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Email == "" || body.Name == "" || body.Password == "" || body.Role == "" {
		writeJSONError(w, http.StatusBadRequest, "email, name, password y role son requeridos")
		return
	}
	if body.Role != "admin" && body.Role != "operator" && body.Role != "viewer" {
		writeJSONError(w, http.StatusBadRequest, "role debe ser admin, operator o viewer")
		return
	}

	hash, err := service.HashPassword(body.Password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al hashear contraseña")
		return
	}

	user := &domain.User{
		Email:        body.Email,
		Name:         body.Name,
		Role:         body.Role,
		PasswordHash: hash,
	}

	id, err := h.UserRepo.Create(r.Context(), user)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// UpdateUser PUT /api/admin/users/{id}
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Email == "" || body.Name == "" || body.Role == "" {
		writeJSONError(w, http.StatusBadRequest, "email, name y role son requeridos")
		return
	}
	if body.Role != "admin" && body.Role != "operator" && body.Role != "viewer" {
		writeJSONError(w, http.StatusBadRequest, "role debe ser admin, operator o viewer")
		return
	}

	user := &domain.User{
		ID:    id,
		Email: body.Email,
		Name:  body.Name,
		Role:  body.Role,
	}

	if err := h.UserRepo.Update(r.Context(), user); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ChangePassword PUT /api/admin/users/{id}/password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "password es requerido")
		return
	}

	hash, err := service.HashPassword(body.Password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al hashear contraseña")
		return
	}

	if err := h.UserRepo.UpdatePassword(r.Context(), id, hash); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeactivateUser DELETE /api/admin/users/{id}
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.UserRepo.Deactivate(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
