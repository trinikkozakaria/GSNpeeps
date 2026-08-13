package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// Insert menulis draft notifikasi secara idempotent dan mengembalikan jumlah baris baru.
//
// `ON CONFLICT DO NOTHING` pada UNIQUE (recipient_user_id, event_key) membuat retry producer,
// dua worker paralel, maupun dua consumer atas event yang sama menghasilkan satu baris.
// Notifikasi yang sudah di-dismiss juga tidak dapat hidup kembali karena barisnya masih ada
// dan konflik tetap terjadi.
//
// Method memakai executor sehingga ikut serta pada transaction pemanggil bila ada.
func (r *NotificationRepository) Insert(
	ctx context.Context,
	drafts []domain.NotificationDraft,
) (int, error) {
	created := 0
	for _, draft := range drafts {
		var referenceType *string
		if draft.ReferenceType != nil {
			value := string(*draft.ReferenceType)
			referenceType = &value
		}
		tag, err := executor(ctx, r.pool).Exec(ctx, `
            INSERT INTO notifications (
                recipient_user_id, tipe, judul, pesan,
                referensi_id, referensi_tipe, event_key, created_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            ON CONFLICT (recipient_user_id, event_key) DO NOTHING
        `,
			draft.RecipientUserID, string(draft.Type), draft.Title, draft.Message,
			draft.ReferenceID, referenceType, draft.EventKey, draft.CreatedAt,
		)
		if err != nil {
			return created, fmt.Errorf("insert notification: %w", err)
		}
		created += int(tag.RowsAffected())
	}
	return created, nil
}

// List mengembalikan inbox satu penerima. Filter penerima selalu diterapkan di query,
// bukan hanya di handler, sehingga tidak ada jalur yang membaca notifikasi orang lain.
func (r *NotificationRepository) List(
	ctx context.Context,
	recipientUserID uuid.UUID,
	isRead *bool,
	page, limit int,
) (domain.NotificationPage, error) {
	result := domain.NotificationPage{Page: page, Limit: limit, Items: []domain.Notification{}}

	err := r.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM notifications
        WHERE recipient_user_id = $1
          AND dismissed_at IS NULL
          AND ($2::BOOLEAN IS NULL OR is_read = $2)
    `, recipientUserID, isRead).Scan(&result.Total)
	if err != nil {
		return domain.NotificationPage{}, fmt.Errorf("count notifications: %w", err)
	}
	if result.Total == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
        SELECT id, recipient_user_id, tipe, judul, pesan,
               referensi_id, referensi_tipe, is_read, read_at, created_at
        FROM notifications
        WHERE recipient_user_id = $1
          AND dismissed_at IS NULL
          AND ($2::BOOLEAN IS NULL OR is_read = $2)
        ORDER BY created_at DESC, id DESC
        LIMIT $3 OFFSET $4
    `, recipientUserID, isRead, limit, (page-1)*limit)
	if err != nil {
		return domain.NotificationPage{}, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.Notification
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Type, &item.Title, &item.Message,
			&item.ReferenceID, &item.ReferenceType, &item.IsRead, &item.ReadAt, &item.CreatedAt,
		); err != nil {
			return domain.NotificationPage{}, fmt.Errorf("scan notification: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.NotificationPage{}, fmt.Errorf("iterate notifications: %w", err)
	}
	return result, nil
}

// UnreadCount memakai partial index idx_notifications_recipient_unread.
func (r *NotificationRepository) UnreadCount(
	ctx context.Context,
	recipientUserID uuid.UUID,
) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM notifications
        WHERE recipient_user_id = $1
          AND dismissed_at IS NULL
          AND is_read = FALSE
    `, recipientUserID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return total, nil
}

// MarkRead menandai notifikasi milik penerima sebagai dibaca. Operasi idempotent: pemanggilan
// berulang tetap berhasil dan tidak menggeser `read_at` yang sudah tercatat.
// ErrNotFound dikembalikan bila notifikasi bukan milik penerima, tidak ada, atau sudah
// di-dismiss; service memetakannya ke satu kode yang seragam.
func (r *NotificationRepository) MarkRead(
	ctx context.Context,
	recipientUserID uuid.UUID,
	id uuid.UUID,
	readAt time.Time,
) error {
	var updated uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
        UPDATE notifications
        SET is_read = TRUE, read_at = COALESCE(read_at, $3)
        WHERE id = $1
          AND recipient_user_id = $2
          AND dismissed_at IS NULL
        RETURNING id
    `, id, recipientUserID, readAt).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

// Dismiss mengisi `dismissed_at`. Tidak ada penghapusan fisik sehingga event yang sama tidak
// dapat dibuat ulang oleh producer.
func (r *NotificationRepository) Dismiss(
	ctx context.Context,
	recipientUserID uuid.UUID,
	id uuid.UUID,
	dismissedAt time.Time,
) error {
	var updated uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
        UPDATE notifications
        SET dismissed_at = COALESCE(dismissed_at, $3)
        WHERE id = $1
          AND recipient_user_id = $2
        RETURNING id
    `, id, recipientUserID, dismissedAt).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("dismiss notification: %w", err)
	}
	return nil
}

// ApproverUserIDs mengembalikan user aktif yang menjadi approver pada satu tahap.
//
// Tahap Atasan menghasilkan paling banyak satu penerima, yaitu atasan langsung pemohon yang
// masih aktif. Tahap HR menghasilkan seluruh HR aktif. Tahap Top Management menghasilkan
// satu-satunya Top Management aktif. Karyawan nonaktif atau terhapus tidak pernah terpilih.
func (r *NotificationRepository) ApproverUserIDs(
	ctx context.Context,
	stage domain.ApprovalStage,
	requesterUserID uuid.UUID,
) ([]uuid.UUID, error) {
	if stage == domain.StageSupervisor {
		return r.collectUserIDs(ctx, `
            SELECT supervisor_user.id
            FROM users requester
            JOIN employees requester_employee ON requester_employee.id = requester.employee_id
            JOIN employees supervisor_employee ON supervisor_employee.id = requester_employee.atasan_id
            JOIN users supervisor_user ON supervisor_user.employee_id = supervisor_employee.id
            WHERE requester.id = $1
              AND supervisor_employee.status = 'aktif'
              AND supervisor_employee.deleted_at IS NULL
        `, requesterUserID)
	}

	role := domain.RoleHR
	if stage == domain.StageTopManagement {
		role = domain.RoleTopManagement
	}
	return r.ActiveUserIDsByRole(ctx, role)
}

// ActiveUserIDsByRole mengembalikan seluruh user aktif pada satu role.
func (r *NotificationRepository) ActiveUserIDsByRole(
	ctx context.Context,
	role domain.RoleName,
) ([]uuid.UUID, error) {
	return r.collectUserIDs(ctx, `
        SELECT users.id
        FROM users
        JOIN roles ON roles.id = users.role_id
        JOIN employees ON employees.id = users.employee_id
        WHERE roles.nama = $1
          AND employees.status = 'aktif'
          AND employees.deleted_at IS NULL
        ORDER BY users.id
    `, string(role))
}

// SupervisorUserID mengembalikan user atasan langsung aktif dari seorang karyawan.
func (r *NotificationRepository) SupervisorUserID(
	ctx context.Context,
	employeeID uuid.UUID,
) (*uuid.UUID, error) {
	ids, err := r.collectUserIDs(ctx, `
        SELECT supervisor_user.id
        FROM employees subordinate
        JOIN employees supervisor_employee ON supervisor_employee.id = subordinate.atasan_id
        JOIN users supervisor_user ON supervisor_user.employee_id = supervisor_employee.id
        WHERE subordinate.id = $1
          AND supervisor_employee.status = 'aktif'
          AND supervisor_employee.deleted_at IS NULL
    `, employeeID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &ids[0], nil
}

// ClaimExpiringContracts mengembalikan kontrak aktif yang berakhir tepat pada tanggal acuan.
// Batch dibatasi agar satu run tidak menahan koneksi terlalu lama pada data besar.
func (r *NotificationRepository) ClaimExpiringContracts(
	ctx context.Context,
	endDate time.Time,
	limit int,
) ([]domain.ExpiringContract, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT contracts.id, contracts.employee_id, users.id, contracts.tanggal_berakhir
        FROM employee_contracts contracts
        JOIN employees ON employees.id = contracts.employee_id
        JOIN users ON users.employee_id = employees.id
        WHERE contracts.status = 'aktif'
          AND contracts.tanggal_berakhir = $1
          AND employees.status = 'aktif'
          AND employees.deleted_at IS NULL
        ORDER BY contracts.id
        LIMIT $2
    `, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("list expiring contracts: %w", err)
	}
	defer rows.Close()

	items := []domain.ExpiringContract{}
	for rows.Next() {
		var item domain.ExpiringContract
		var endsAt time.Time
		if err := rows.Scan(&item.ContractID, &item.EmployeeID, &item.UserID, &endsAt); err != nil {
			return nil, fmt.Errorf("scan expiring contract: %w", err)
		}
		item.EndDate = endsAt.Format(domain.DateLayout)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expiring contracts: %w", err)
	}
	return items, nil
}

func (r *NotificationRepository) collectUserIDs(
	ctx context.Context,
	query string,
	args ...any,
) ([]uuid.UUID, error) {
	rows, err := executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("resolve notification recipients: %w", err)
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan notification recipient: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification recipients: %w", err)
	}
	return ids, nil
}
