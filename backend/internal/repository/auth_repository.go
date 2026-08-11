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

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

const accountColumns = `
    u.id,
    u.employee_id,
    u.role_id,
    e.nama,
    u.email,
    u.password_hash,
    r.nama,
    e.status,
    e.deleted_at,
    u.failed_login_count,
    u.account_locked,
    e.foto_profil_url`

func (r *AuthRepository) FindForLogin(ctx context.Context, email string) (domain.LoginAccount, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT `+accountColumns+`
        FROM users u
        JOIN employees e ON e.id = u.employee_id
        JOIN roles r ON r.id = u.role_id
        WHERE u.email = $1
    `, email)
	return scanAccount(row)
}

func (r *AuthRepository) FindForPasswordByID(ctx context.Context, userID uuid.UUID) (domain.LoginAccount, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT `+accountColumns+`
        FROM users u
        JOIN employees e ON e.id = u.employee_id
        JOIN roles r ON r.id = u.role_id
        WHERE u.id = $1
    `, userID)
	return scanAccount(row)
}

func (r *AuthRepository) FindIdentityByID(ctx context.Context, userID uuid.UUID) (domain.AuthUser, error) {
	var user domain.AuthUser
	err := r.pool.QueryRow(ctx, `
        SELECT u.id, u.employee_id, e.nama, u.email, r.nama, e.foto_profil_url
        FROM users u
        JOIN employees e ON e.id = u.employee_id
        JOIN roles r ON r.id = u.role_id
        WHERE u.id = $1
          AND e.status = 'aktif'
          AND e.deleted_at IS NULL
    `, userID).Scan(&user.ID, &user.EmployeeID, &user.Name, &user.Email, &user.Role, &user.PhotoURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthUser{}, ErrNotFound
	}
	if err != nil {
		return domain.AuthUser{}, fmt.Errorf("find auth identity: %w", err)
	}
	return user, nil
}

func (r *AuthRepository) RecordFailedLogin(
	ctx context.Context,
	userID uuid.UUID,
	threshold int,
) (int, bool, error) {
	var count int
	var locked bool
	err := r.pool.QueryRow(ctx, `
        UPDATE users
        SET failed_login_count = failed_login_count + 1,
            account_locked = (failed_login_count + 1 >= $2),
            updated_at = NOW()
        WHERE id = $1
        RETURNING failed_login_count, account_locked
    `, userID, threshold).Scan(&count, &locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, ErrNotFound
	}
	if err != nil {
		return 0, false, fmt.Errorf("record failed login: %w", err)
	}
	return count, locked, nil
}

func (r *AuthRepository) RecordSuccessfulLogin(ctx context.Context, userID uuid.UUID, at time.Time) error {
	command, err := r.pool.Exec(ctx, `
        UPDATE users
        SET failed_login_count = 0,
            account_locked = FALSE,
            last_login_at = $2,
            updated_at = NOW()
        WHERE id = $1
    `, userID, at)
	if err != nil {
		return fmt.Errorf("record successful login: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *AuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	command, err := r.pool.Exec(ctx, `
        UPDATE users
        SET password_hash = $2,
            failed_login_count = 0,
            account_locked = FALSE,
            updated_at = NOW()
        WHERE id = $1
    `, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row rowScanner) (domain.LoginAccount, error) {
	var account domain.LoginAccount
	err := row.Scan(
		&account.ID,
		&account.EmployeeID,
		&account.RoleID,
		&account.Name,
		&account.Email,
		&account.PasswordHash,
		&account.Role,
		&account.EmployeeStatus,
		&account.EmployeeDeleted,
		&account.FailedLoginCount,
		&account.AccountLocked,
		&account.PhotoURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LoginAccount{}, ErrNotFound
	}
	if err != nil {
		return domain.LoginAccount{}, fmt.Errorf("scan login account: %w", err)
	}
	return account, nil
}
