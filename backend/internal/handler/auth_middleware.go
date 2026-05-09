package handler

import (
	"context"
	"net/http"
	"strings"

	"pasaje/backend/internal/service"
)

type contextKey string

const claimsKey contextKey = "claims"

// AuthMiddleware valida el token JWT en el header Authorization.
func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			var tokenStr string

			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenStr = parts[1]
				}
			}

			// Fallback: accept token from query parameter (needed for <img src=""> requests)
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				writeJSONError(w, http.StatusUnauthorized, "token requerido")
				return
			}

			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "token inválido o expirado")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extrae los claims del contexto.
func UserFromContext(ctx context.Context) *service.Claims {
	claims, _ := ctx.Value(claimsKey).(*service.Claims)
	return claims
}
