package domain

import "time"

// User representa un usuario del sistema.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"` // admin | operator | viewer
	AgencyID     *int64    `json:"agency_id,omitempty"`
	Active       bool      `json:"active"`
	PasswordHash string    `json:"-"`              // never expose in JSON
	GoogleID     *string   `json:"google_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LoginRequest es el body para POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse es la respuesta de login exitoso.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateUserRequest es el body para POST /api/admin/users.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}
