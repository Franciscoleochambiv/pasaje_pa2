package handler

import (
	"encoding/json"
	"net/http"

	"pasaje/backend/internal/domain"
	"pasaje/backend/internal/service"
)

// GoogleAuthHandler maneja los endpoints de autenticación con Google OAuth.
type GoogleAuthHandler struct {
	GoogleAuth *service.GoogleAuthService
}

// LoginWithGoogle POST /api/auth/google
func (h *GoogleAuthHandler) LoginWithGoogle(w http.ResponseWriter, r *http.Request) {
	var body service.GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Credential == "" {
		writeJSONError(w, http.StatusBadRequest, "credential es requerido")
		return
	}

	token, user, err := h.GoogleAuth.LoginWithGoogle(r.Context(), body.Credential)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, domain.LoginResponse{
		Token: token,
		User:  *user,
	})
}
