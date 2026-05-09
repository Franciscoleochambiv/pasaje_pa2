package handler

import (
	"encoding/json"
	"net/http"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/service"
)

// AuthHandler maneja los endpoints /api/auth/*.
type AuthHandler struct {
	Auth *service.AuthService
}

// Login POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "email y password son requeridos")
		return
	}

	token, user, err := h.Auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, domain.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// Me GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := UserFromContext(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}
