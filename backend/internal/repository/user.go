package repository

import (
	"context"

	"pasaje/backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository acceso a datos de usuarios.
type UserRepository struct {
	Pool *pgxpool.Pool
}

// GetByEmail devuelve un usuario activo por email (incluye password_hash para login).
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, agency_id, active, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1 AND active = true
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.AgencyID, &u.Active, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// List devuelve todos los usuarios (sin password_hash).
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, email, name, role, agency_id, active, created_at, updated_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.AgencyID, &u.Active, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// Create inserta un usuario y devuelve su ID.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name, role, agency_id, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, u.Email, u.Name, u.Role, u.AgencyID, u.PasswordHash).Scan(&id)
	return id, err
}

// Update actualiza nombre, email y rol de un usuario.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE users SET email = $2, name = $3, role = $4, updated_at = now()
		WHERE id = $1
	`, u.ID, u.Email, u.Name, u.Role)
	return err
}

// UpdatePassword actualiza solo el password_hash de un usuario.
func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = now()
		WHERE id = $1
	`, id, hash)
	return err
}

// Deactivate desactiva un usuario (soft delete).
func (r *UserRepository) Deactivate(ctx context.Context, id int64) error {
	_, err := r.Pool.Exec(ctx, `
		UPDATE users SET active = false, updated_at = now()
		WHERE id = $1
	`, id)
	return err
}
