package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pasaje/backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GoogleAuthService handles customer authentication via Google OAuth.
type GoogleAuthService struct {
	Pool            *pgxpool.Pool
	GoogleClientID  string
	GoogleClientIDs []string // all allowed audience values (web + mobile)
	JWTSecret       string
}

// GoogleLoginRequest is the request from frontend after Google Sign-In.
type GoogleLoginRequest struct {
	Credential string `json:"credential"` // Google ID token from frontend
}

// googleTokenInfo represents the response from Google's tokeninfo endpoint.
type googleTokenInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Sub           string `json:"sub"` // Google user ID
	Aud           string `json:"aud"`
	EmailVerified string `json:"email_verified"`
}

// LoginWithGoogle verifies a Google ID token, creates or finds the customer user,
// and returns a JWT token along with the user.
func (s *GoogleAuthService) LoginWithGoogle(ctx context.Context, idToken string) (string, *domain.User, error) {
	// 1. Verify Google ID token via tokeninfo endpoint
	info, err := s.verifyGoogleToken(idToken)
	if err != nil {
		return "", nil, fmt.Errorf("token de Google inválido: %w", err)
	}

	// 2. Verify that aud matches one of our allowed GoogleClientIDs
	if !s.isAllowedAudience(info.Aud) {
		return "", nil, errors.New("token de Google no corresponde a esta aplicación")
	}

	// 3. Verify email is verified
	if info.EmailVerified != "true" {
		return "", nil, errors.New("email de Google no verificado")
	}

	// 4. Find or create customer user
	user, err := s.findOrCreateCustomer(ctx, info)
	if err != nil {
		return "", nil, fmt.Errorf("creando usuario: %w", err)
	}

	// 5. Generate JWT
	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generando token: %w", err)
	}

	return token, user, nil
}

func (s *GoogleAuthService) verifyGoogleToken(idToken string) (*googleTokenInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("verificando token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("token de Google rechazado")
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("parseando respuesta de Google: %w", err)
	}

	if info.Email == "" || info.Sub == "" {
		return nil, errors.New("respuesta de Google incompleta")
	}

	return &info, nil
}

func (s *GoogleAuthService) findOrCreateCustomer(ctx context.Context, info *googleTokenInfo) (*domain.User, error) {
	// Try to find by google_id first
	var user domain.User
	err := s.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, agency_id, active, password_hash, google_id, created_at, updated_at
		FROM users
		WHERE google_id = $1 AND active = true
	`, info.Sub).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.AgencyID, &user.Active, &user.PasswordHash, &user.GoogleID, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		user.PasswordHash = ""
		return &user, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Try to find by email (may have been created without google_id)
	err = s.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, agency_id, active, password_hash, google_id, created_at, updated_at
		FROM users
		WHERE email = $1 AND active = true
	`, info.Email).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.AgencyID, &user.Active, &user.PasswordHash, &user.GoogleID, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		// Link Google ID to existing user
		_, _ = s.Pool.Exec(ctx, `UPDATE users SET google_id = $2, updated_at = now() WHERE id = $1`, user.ID, info.Sub)
		user.GoogleID = &info.Sub
		user.PasswordHash = ""
		return &user, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new customer user
	googleID := info.Sub
	var id int64
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name, role, password_hash, google_id, active)
		VALUES ($1, $2, 'customer', '', $3, true)
		RETURNING id
	`, info.Email, info.Name, googleID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insertando usuario: %w", err)
	}

	return &domain.User{
		ID:       id,
		Email:    info.Email,
		Name:     info.Name,
		Role:     "customer",
		Active:   true,
		GoogleID: &googleID,
	}, nil
}

func (s *GoogleAuthService) generateToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.JWTSecret))
}

// isAllowedAudience checks if the token audience matches any configured client ID.
func (s *GoogleAuthService) isAllowedAudience(aud string) bool {
	if len(s.GoogleClientIDs) == 0 && s.GoogleClientID == "" {
		return true // no restriction configured
	}
	for _, id := range s.GoogleClientIDs {
		if aud == id {
			return true
		}
	}
	// Fallback to legacy single ID
	if s.GoogleClientID != "" && aud == s.GoogleClientID {
		return true
	}
	return false
}
