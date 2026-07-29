package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Append(ctx context.Context, entry domain.AuditEntry) error {
	detail, err := json.Marshal(entry.Detail)
	if err != nil {
		return fmt.Errorf("marshal audit detail: %w", err)
	}
	_, err = executor(ctx, r.pool).Exec(ctx, `
        INSERT INTO audit_logs (user_id, aksi, modul, data_id, detail, ip_address, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, entry.UserID, entry.Action, entry.Module, entry.DataID, detail, entry.IPAddress, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("append audit log: %w", err)
	}
	return nil
}
