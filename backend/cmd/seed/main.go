package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/password"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
)

type fixture struct {
	NIP        string
	Name       string
	Email      string
	Gender     string
	Role       string
	Supervisor string
}

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.App.Environment != "development" && cfg.App.Environment != "test" {
		return errors.New("synthetic auth seed is allowed only in development or test")
	}
	seedPassword := os.Getenv("SEED_PASSWORD")
	if len(seedPassword) < 12 {
		return errors.New("SEED_PASSWORD must contain at least 12 characters")
	}
	hasher := password.New(cfg.Auth)
	passwordHash, err := hasher.Hash(seedPassword)
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := postgres.Open(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close()

	transaction, err := db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer transaction.Rollback(ctx)

	var departmentID uuid.UUID
	err = transaction.QueryRow(ctx, `
        INSERT INTO departments (nama)
        VALUES ('Departemen Sintetis')
        ON CONFLICT (nama) DO UPDATE SET updated_at = NOW()
        RETURNING id
    `).Scan(&departmentID)
	if err != nil {
		return fmt.Errorf("seed department: %w", err)
	}

	var positionID uuid.UUID
	err = transaction.QueryRow(ctx, `
        INSERT INTO positions (nama, department_id)
        VALUES ('Posisi Sintetis', $1)
        ON CONFLICT (department_id, nama) DO UPDATE SET updated_at = NOW()
        RETURNING id
    `, departmentID).Scan(&positionID)
	if err != nil {
		return fmt.Errorf("seed position: %w", err)
	}

	// Koordinat sintetis ini hanya untuk pengujian aturan radius WFO. Nilainya tidak
	// merepresentasikan kantor nyata dan seed sendiri ditolak di luar development/test.
	if _, err := transaction.Exec(ctx, `
        INSERT INTO office_locations (kode, nama, alamat, latitude, longitude, is_active)
        VALUES ('OFFICE-SYN-001', 'Kantor Sintetis', 'Alamat kantor sintetis', -6.2000000, 106.8000000, TRUE)
        ON CONFLICT (kode) DO UPDATE
        SET nama = EXCLUDED.nama,
            alamat = EXCLUDED.alamat,
            latitude = EXCLUDED.latitude,
            longitude = EXCLUDED.longitude,
            is_active = TRUE,
            updated_at = NOW()
    `); err != nil {
		return fmt.Errorf("seed office location: %w", err)
	}

	// Dua master izin sintetis memisahkan alur approval tanpa kuota dari validasi
	// dokumen wajib. Keduanya tidak merepresentasikan kebijakan cuti produksi.
	if _, err := transaction.Exec(ctx, `
        INSERT INTO leave_types (kode, nama, kuota_tahunan, memerlukan_dokumen, is_active)
        VALUES
            ('IZIN-SYN-E2E', 'Izin Sintetis Tanpa Kuota', 0, FALSE, TRUE),
            ('CUTI-SYN-E2E', 'Cuti Sintetis Wajib Dokumen', 12, TRUE, TRUE)
        ON CONFLICT (kode) DO UPDATE
        SET nama = EXCLUDED.nama,
            kuota_tahunan = EXCLUDED.kuota_tahunan,
            memerlukan_dokumen = EXCLUDED.memerlukan_dokumen,
            is_active = TRUE,
            updated_at = NOW()
    `); err != nil {
		return fmt.Errorf("seed leave types: %w", err)
	}

	fixtures := []fixture{
		{NIP: "SYN-ATASAN-001", Name: "Atasan Sintetis", Email: "atasan@example.test", Gender: "L", Role: "atasan"},
		{NIP: "SYN-KARYAWAN-001", Name: "Karyawan Sintetis", Email: "karyawan@example.test", Gender: "P", Role: "karyawan", Supervisor: "SYN-ATASAN-001"},
		{NIP: "SYN-KARYAWAN-002", Name: "Karyawan Tanpa Atasan Sintetis", Email: "karyawan.tanpa.atasan@example.test", Gender: "L", Role: "karyawan"},
		{NIP: "SYN-HR-001", Name: "HR Sintetis", Email: "hr@example.test", Gender: "P", Role: "hr"},
		{NIP: "SYN-TM-001", Name: "Top Management Sintetis", Email: "top.management@example.test", Gender: "L", Role: "top_management"},
	}

	employeeIDs := make(map[string]uuid.UUID, len(fixtures))
	for _, item := range fixtures {
		var employeeID uuid.UUID
		err := transaction.QueryRow(ctx, `
            INSERT INTO employees (
                nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join,
                department_id, position_id, status
            )
            VALUES ($1, $2, $3, DATE '1990-01-01', CURRENT_DATE, $4, $5, 'aktif')
            ON CONFLICT (nip) DO UPDATE
            SET nama = EXCLUDED.nama,
                jenis_kelamin = EXCLUDED.jenis_kelamin,
                department_id = EXCLUDED.department_id,
                position_id = EXCLUDED.position_id,
                status = 'aktif',
                deleted_at = NULL,
                updated_at = NOW()
            RETURNING id
        `, item.NIP, item.Name, item.Gender, departmentID, positionID).Scan(&employeeID)
		if err != nil {
			return fmt.Errorf("seed employee %s: %w", item.NIP, err)
		}
		employeeIDs[item.NIP] = employeeID
	}

	for _, item := range fixtures {
		if item.Supervisor != "" {
			if _, err := transaction.Exec(ctx, `
                UPDATE employees SET atasan_id = $2, updated_at = NOW() WHERE id = $1
            `, employeeIDs[item.NIP], employeeIDs[item.Supervisor]); err != nil {
				return fmt.Errorf("seed supervisor relation: %w", err)
			}
		}
		roleID, err := roleID(ctx, transaction, item.Role)
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
            INSERT INTO users (id, employee_id, email, password_hash, role_id)
            VALUES (
                COALESCE(
                    (SELECT id FROM users WHERE employee_id = $1),
                    gen_random_uuid()
                ),
                $1, $2, $3, $4
            )
            ON CONFLICT (employee_id) DO UPDATE
            SET email = EXCLUDED.email,
                password_hash = EXCLUDED.password_hash,
                role_id = EXCLUDED.role_id,
                failed_login_count = 0,
                account_locked = FALSE,
                updated_at = NOW()
        `, employeeIDs[item.NIP], strings.ToLower(item.Email), passwordHash, roleID); err != nil {
			return fmt.Errorf("seed user %s: %w", item.Email, err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	slog.Info("synthetic auth fixtures seeded", "accounts", len(fixtures))
	return nil
}

func roleID(ctx context.Context, transaction pgx.Tx, name string) (uuid.UUID, error) {
	var id uuid.UUID
	if err := transaction.QueryRow(ctx, `SELECT id FROM roles WHERE nama = $1`, name).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("find role %s: %w", name, err)
	}
	return id, nil
}
