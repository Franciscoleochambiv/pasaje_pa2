package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepository struct {
	Pool *pgxpool.Pool
}

type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func (r *SettingsRepository) GetAll(ctx context.Context) ([]Setting, error) {
	rows, err := r.Pool.Query(ctx, `SELECT key, value, COALESCE(description,'') FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.Description); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := r.Pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&val)
	return val, err
}

func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now()
	`, key, value)
	return err
}

func (r *SettingsRepository) SetBulk(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		if err := r.Set(ctx, k, v); err != nil {
			return err
		}
	}
	return nil
}
