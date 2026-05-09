package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func truncateForLog(s string) string {
	const maxLen = 400
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func writeVoidAudit(ctx context.Context, pool *pgxpool.Pool, entity string, entityID *int64, action string, actorID *int64, payload map[string]any) {
	if pool == nil {
		return
	}
	var data []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			log.Printf("void-audit: marshal payload entity=%s action=%s err=%v", entity, action, err)
			return
		}
		data = b
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO audit_logs (entity, entity_id, action, actor_id, new_data)
		VALUES ($1, $2, $3, $4, $5)
	`, entity, entityID, action, actorID, data)
	if err != nil {
		log.Printf("void-audit: insert entity=%s action=%s err=%v", entity, action, err)
	}
}
