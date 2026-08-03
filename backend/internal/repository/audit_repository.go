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

// List membaca audit log dengan filter kontrak. Repository ini sengaja hanya memiliki Append
// dan List: tidak ada method update maupun delete, dan database menolak keduanya melalui
// trigger append-only pada migration 00006.
func (r *AuditRepository) List(
	ctx context.Context,
	filter domain.AuditLogFilter,
) (domain.AuditLogPage, error) {
	result := domain.AuditLogPage{
		Page:  filter.Page,
		Limit: filter.Limit,
		Items: []domain.AuditLogEntry{},
	}

	// Seluruh filter dikirim sebagai parameter; tidak ada fragmen SQL yang dibentuk dari
	// input pengguna.
	arguments := []any{
		filter.UserID, filter.Action, filter.Module, filter.StartDate, filter.EndDate,
	}
	const conditions = `
        WHERE ($1::UUID IS NULL OR audit_logs.user_id = $1)
          AND ($2::TEXT IS NULL OR audit_logs.aksi = $2)
          AND ($3::TEXT IS NULL OR audit_logs.modul = $3)
          AND ($4::TIMESTAMP IS NULL OR audit_logs.created_at >= $4)
          AND ($5::TIMESTAMP IS NULL OR audit_logs.created_at < $5)
    `

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs`+conditions, arguments...,
	).Scan(&result.Total); err != nil {
		return domain.AuditLogPage{}, fmt.Errorf("count audit logs: %w", err)
	}
	if result.Total == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
        SELECT audit_logs.id, audit_logs.user_id, employees.nama, audit_logs.aksi,
               audit_logs.modul, audit_logs.data_id, audit_logs.detail,
               audit_logs.ip_address, audit_logs.created_at
        FROM audit_logs
        LEFT JOIN users ON users.id = audit_logs.user_id
        LEFT JOIN employees ON employees.id = users.employee_id
    `+conditions+`
        ORDER BY audit_logs.created_at DESC, audit_logs.id DESC
        LIMIT $6 OFFSET $7
    `, append(arguments, filter.Limit, (filter.Page-1)*filter.Limit)...)
	if err != nil {
		return domain.AuditLogPage{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry domain.AuditLogEntry
		var detail []byte
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.UserName, &entry.Action, &entry.Module,
			&entry.ResourceID, &detail, &entry.IPAddress, &entry.CreatedAt,
		); err != nil {
			return domain.AuditLogPage{}, fmt.Errorf("scan audit log: %w", err)
		}
		if len(detail) > 0 {
			if err := json.Unmarshal(detail, &entry.Detail); err != nil {
				return domain.AuditLogPage{}, fmt.Errorf("decode audit detail: %w", err)
			}
		}
		result.Items = append(result.Items, entry)
	}
	if err := rows.Err(); err != nil {
		return domain.AuditLogPage{}, fmt.Errorf("iterate audit logs: %w", err)
	}
	return result, nil
}
