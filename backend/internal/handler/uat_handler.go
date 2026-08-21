package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/middleware"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/pkg/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultFeedPageSize mengikuti kebutuhan infinite scroll Beranda (20 feed per load).
const defaultFeedPageSize = 20

// maxFeedPageSize membatasi limit agar klien tidak dapat meminta dataset tak terbatas.
const maxFeedPageSize = 100

// UATHandler menangani master dan konten sederhana yang tetap disimpan PostgreSQL.
// Aturan role diterapkan di sini selain penyaringan menu di frontend.
type mediaStore interface {
	Upload(context.Context, string, io.Reader, string) (string, error)
	Download(context.Context, string) (io.ReadCloser, string, error)
	Delete(context.Context, string) error
}
type UATHandler struct {
	db    *pgxpool.Pool
	media mediaStore
}

func NewUATHandler(db *pgxpool.Pool, media ...mediaStore) *UATHandler {
	h := &UATHandler{db: db}
	if len(media) > 0 {
		h.media = media[0]
	}
	return h
}

func (h *UATHandler) Media(w http.ResponseWriter, r *http.Request) {
	if h.media == nil {
		response.Error(w, 503, "SERVICE_UNAVAILABLE", "Berkas belum tersedia")
		return
	}
	stored := strings.TrimSpace(r.URL.Query().Get("path"))
	if stored == "" {
		response.Error(w, 400, "INVALID_PARAM", "Path berkas tidak valid")
		return
	}
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Sesi tidak valid")
		return
	}
	allowed := identity.Role == domain.RoleHR
	if strings.Contains(stored, "/employee-photos/"+identity.EmployeeID.String()+"/") ||
		strings.Contains(stored, "/attendance-photos/"+identity.UserID.String()+"/") ||
		strings.Contains(stored, "/leave-documents/"+identity.UserID.String()+"/") ||
		strings.Contains(stored, "/overtime-documents/"+identity.UserID.String()+"/") {
		allowed = true
	}
	if strings.Contains(stored, "/leave-documents/") && (identity.Role == domain.RoleSupervisor || identity.Role == domain.RoleTopManagement) {
		allowed = true
	}
	if strings.Contains(stored, "/overtime-documents/") && (identity.Role == domain.RoleSupervisor || identity.Role == domain.RoleHR || identity.Role == domain.RoleTopManagement) {
		allowed = true
	}
	if strings.HasPrefix(stored, "company-feed/") || strings.Contains(stored, "/company-feed/") {
		allowed = true
	}
	if !allowed {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses ke berkas")
		return
	}
	body, contentType, err := h.media.Download(r.Context(), stored)
	if err != nil {
		response.Error(w, 404, "NOT_FOUND", "Berkas tidak ditemukan")
		return
	}
	defer body.Close()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(200)
	_, _ = io.Copy(w, body)
}

func (h *UATHandler) HomeSummary(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		return
	}
	balances := make([]map[string]any, 0)
	rows, err := h.db.Query(r.Context(), `SELECT COALESCE(lt.nama,'Cuti tahunan'),lb.saldo_awal,lb.saldo_terpakai,lb.saldo_sisa FROM leave_balances lb LEFT JOIN leave_types lt ON lt.id=lb.leave_type_id WHERE lb.user_id=$1 AND lb.tahun=EXTRACT(YEAR FROM NOW()) ORDER BY lt.nama NULLS FIRST`, identity.UserID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var opening, used, remaining int
			if rows.Scan(&name, &opening, &used, &remaining) == nil {
				balances = append(balances, map[string]any{"jenis": name, "saldo_awal": opening, "terpakai": used, "sisa": remaining})
			}
		}
	}
	var mine int
	_ = h.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM leave_requests WHERE user_id=$1 AND status NOT IN ('dibatalkan')`, identity.UserID).Scan(&mine)
	pending := 0
	switch identity.Role {
	case domain.RoleSupervisor:
		_ = h.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM leave_requests lr JOIN users u ON u.id=lr.user_id JOIN employees e ON e.id=u.employee_id WHERE e.atasan_id=$1 AND lr.status='menunggu_atasan'`, identity.EmployeeID).Scan(&pending)
	case domain.RoleHR:
		_ = h.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM leave_requests WHERE status='menunggu_hr')+(SELECT COUNT(*) FROM overtime_requests WHERE status='menunggu_hr')+(SELECT COUNT(*) FROM attendance_corrections WHERE status='menunggu_hr')`).Scan(&pending)
	case domain.RoleTopManagement:
		_ = h.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM leave_requests WHERE status='menunggu_top_management')+(SELECT COUNT(*) FROM overtime_requests WHERE status='menunggu_top_management')`).Scan(&pending)
	}
	response.Success(w, 200, map[string]any{"saldo_cuti": balances, "pengajuan_perlu_disetujui": pending, "pengajuan_ketidakhadiran_pribadi": mine}, "Ringkasan beranda berhasil dimuat")
}

type documentTypeInput struct {
	Code     string `json:"kode"`
	Name     string `json:"nama"`
	Required bool   `json:"wajib"`
	IsActive *bool  `json:"is_active"`
}

func requireHR(w http.ResponseWriter, r *http.Request) (domain.Identity, bool) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok || identity.Role != domain.RoleHR {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses")
		return domain.Identity{}, false
	}
	return identity, true
}

func (h *UATHandler) ListDocumentTypes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `SELECT id, kode, nama, wajib, is_active FROM document_types ORDER BY nama`)
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Jenis dokumen belum dapat dimuat")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var code, name string
		var required, active bool
		if rows.Scan(&id, &code, &name, &required, &active) != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Jenis dokumen belum dapat dimuat")
			return
		}
		items = append(items, map[string]any{"id": id, "kode": code, "nama": name, "wajib": required, "is_active": active})
	}
	response.Success(w, 200, items, "Jenis dokumen berhasil dimuat")
}

func (h *UATHandler) CreateDocumentType(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireHR(w, r); !ok {
		return
	}
	var input documentTypeInput
	if decodeJSON(r, &input) != nil || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" {
		response.Error(w, 400, "INVALID_PARAM", "Kode dan nama wajib diisi")
		return
	}
	var id uuid.UUID
	err := h.db.QueryRow(r.Context(), `INSERT INTO document_types (kode,nama,wajib) VALUES ($1,$2,$3) RETURNING id`, strings.TrimSpace(input.Code), strings.TrimSpace(input.Name), input.Required).Scan(&id)
	if err != nil {
		response.Error(w, 409, "CONFLICT", "Kode atau nama jenis dokumen sudah digunakan")
		return
	}
	response.Success(w, 201, map[string]any{"id": id}, "Jenis dokumen berhasil ditambahkan")
}

type feedInput struct {
	Title string `json:"judul"`
	HTML  string `json:"konten_html"`
}

var unsafeHTML = regexp.MustCompile(`(?is)</?(script|iframe|object|embed|style|link|meta|base|form|svg|math)\b|\son\w+\s*=|javascript:|data:text/html`)

// ListFeeds mendukung pagination: Beranda memanggilnya berulang dengan page yang bertambah
// untuk infinite scroll (20 per load), sementara halaman Company Feed memakai kontrol
// pagination biasa dengan limit yang sama.
func (h *UATHandler) ListFeeds(w http.ResponseWriter, r *http.Request) {
	page, ok := positiveIntQuery(w, r, "page", 1)
	if !ok {
		return
	}
	limit, ok := positiveIntQuery(w, r, "limit", defaultFeedPageSize)
	if !ok {
		return
	}
	if limit > maxFeedPageSize {
		limit = maxFeedPageSize
	}

	var total int
	if err := h.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM company_feeds`).Scan(&total); err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat dimuat")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT f.id,f.judul,f.konten_html,f.published_at,e.nama,
		       COALESCE((
		           SELECT jsonb_agg(jsonb_build_object(
		               'id', a.id, 'file_name', a.file_name, 'media_type', a.media_type,
		               'file_size', a.file_size, 'file_url', a.stored_path
		           ) ORDER BY a.created_at)
		           FROM company_feed_attachments a WHERE a.feed_id = f.id
		       ), '[]'::jsonb)
		FROM company_feeds f
		JOIN users u ON u.id=f.author_id
		JOIN employees e ON e.id=u.employee_id
		ORDER BY f.published_at DESC
		LIMIT $1 OFFSET $2
	`, limit, (page-1)*limit)
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat dimuat")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var title, html, author string
		var published time.Time
		var attachmentsJSON []byte
		if rows.Scan(&id, &title, &html, &published, &author, &attachmentsJSON) != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat dimuat")
			return
		}
		attachments := make([]map[string]any, 0)
		if err := json.Unmarshal(attachmentsJSON, &attachments); err != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat dimuat")
			return
		}
		items = append(items, map[string]any{"id": id, "judul": title, "konten_html": html, "published_at": published, "penulis": author, "attachments": attachments})
	}
	response.Paginated(w, items, response.PaginationMeta{
		Page: page, Limit: limit, TotalData: total, TotalPage: totalPages(total, limit),
	}, "Company feed berhasil dimuat")
}

func (h *UATHandler) CreateFeed(w http.ResponseWriter, r *http.Request) {
	identity, ok := requireHR(w, r)
	if !ok {
		return
	}
	var input feedInput
	if decodeJSON(r, &input) != nil || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.HTML) == "" || unsafeHTML.MatchString(input.HTML) {
		response.Error(w, 400, "INVALID_PARAM", "Judul atau konten tidak valid")
		return
	}
	id, err := h.withFeedAudit(w, r, "CREATE", func(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
		var created uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO company_feeds(author_id,judul,konten_html) VALUES($1,$2,$3) RETURNING id`,
			identity.UserID, strings.TrimSpace(input.Title), strings.TrimSpace(input.HTML)).Scan(&created)
		return created, err
	}, "Company feed belum dapat diterbitkan")
	if err != nil {
		return
	}
	response.Success(w, 201, map[string]any{"id": id}, "Company feed berhasil diterbitkan")
}

// UpdateFeed mengubah judul/konten. HR mana pun boleh menyunting feed siapa pun karena
// Company Feed adalah kanal komunikasi bersama, bukan konten milik perorangan.
func (h *UATHandler) UpdateFeed(w http.ResponseWriter, r *http.Request) {
	_, ok := requireHR(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.Error(w, 400, "INVALID_PARAM", "ID tidak valid")
		return
	}
	var input feedInput
	if decodeJSON(r, &input) != nil || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.HTML) == "" || unsafeHTML.MatchString(input.HTML) {
		response.Error(w, 400, "INVALID_PARAM", "Judul atau konten tidak valid")
		return
	}
	_, err = h.withFeedAudit(w, r, "UPDATE", func(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
		tag, execErr := tx.Exec(ctx, `UPDATE company_feeds SET judul=$2,konten_html=$3,updated_at=NOW() WHERE id=$1`,
			id, strings.TrimSpace(input.Title), strings.TrimSpace(input.HTML))
		if execErr == nil && tag.RowsAffected() == 0 {
			execErr = pgx.ErrNoRows
		}
		return id, execErr
	}, "")
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, 404, "NOT_FOUND", "Company feed tidak ditemukan")
		return
	}
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat diperbarui")
		return
	}
	response.Success(w, 200, map[string]any{"id": id}, "Company feed berhasil diperbarui")
}

// DeleteFeed menghapus feed. Company feed bukan salah satu entitas hard-delete-terlarang di
// CLAUDE.md (bukan employee/notifikasi/audit log) dan tidak memiliki kolom soft-delete.
func (h *UATHandler) DeleteFeed(w http.ResponseWriter, r *http.Request) {
	_, ok := requireHR(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.Error(w, 400, "INVALID_PARAM", "ID tidak valid")
		return
	}
	storedPaths := make([]string, 0)
	rows, err := h.db.Query(r.Context(), `SELECT stored_path FROM company_feed_attachments WHERE feed_id=$1`, id)
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Lampiran company feed belum dapat diperiksa")
		return
	}
	for rows.Next() {
		var storedPath string
		if err := rows.Scan(&storedPath); err != nil {
			rows.Close()
			response.Error(w, 500, "INTERNAL_ERROR", "Lampiran company feed belum dapat diperiksa")
			return
		}
		storedPaths = append(storedPaths, storedPath)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Lampiran company feed belum dapat diperiksa")
		return
	}
	_, err = h.withFeedAudit(w, r, "DELETE", func(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
		tag, execErr := tx.Exec(ctx, `DELETE FROM company_feeds WHERE id=$1`, id)
		if execErr == nil && tag.RowsAffected() == 0 {
			execErr = pgx.ErrNoRows
		}
		return id, execErr
	}, "")
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, 404, "NOT_FOUND", "Company feed tidak ditemukan")
		return
	}
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Company feed belum dapat dihapus")
		return
	}
	if h.media != nil {
		for _, storedPath := range storedPaths {
			if cleanupErr := h.media.Delete(r.Context(), storedPath); cleanupErr != nil {
				slog.WarnContext(r.Context(), "company feed attachment cleanup failed", "path", storedPath, "error", cleanupErr)
			}
		}
	}
	response.Success(w, 200, map[string]any{"id": id}, "Company feed berhasil dihapus")
}

// withFeedAudit membungkus mutasi company_feeds dan baris audit_logs dalam satu transaction,
// sama seperti pola create+audit atomik pada service layer lain: kegagalan audit membatalkan
// mutasi alih-alih diam-diam kehilangan jejaknya. errorMessage kosong berarti pemanggil sendiri
// yang memetakan error (mis. NOT_FOUND vs INTERNAL_ERROR).
func (h *UATHandler) withFeedAudit(
	w http.ResponseWriter,
	r *http.Request,
	action string,
	mutate func(context.Context, pgx.Tx) (uuid.UUID, error),
	errorMessage string,
) (uuid.UUID, error) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Sesi tidak valid")
		return uuid.Nil, errors.New("missing identity")
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		if errorMessage != "" {
			response.Error(w, 500, "INTERNAL_ERROR", errorMessage)
		}
		return uuid.Nil, err
	}
	defer tx.Rollback(r.Context())

	id, err := mutate(r.Context(), tx)
	if err != nil {
		if errorMessage != "" && !errors.Is(err, pgx.ErrNoRows) {
			response.Error(w, 500, "INTERNAL_ERROR", errorMessage)
		}
		return id, err
	}
	if _, err = tx.Exec(r.Context(),
		`INSERT INTO audit_logs(user_id,aksi,modul,data_id) VALUES($1,$2,'company_feed',$3)`,
		identity.UserID, action, id,
	); err != nil {
		slog.ErrorContext(r.Context(), "company feed audit write failed", "action", action, "error", err)
		if errorMessage != "" {
			response.Error(w, 500, "INTERNAL_ERROR", errorMessage)
		}
		return id, err
	}
	if err = tx.Commit(r.Context()); err != nil {
		if errorMessage != "" {
			response.Error(w, 500, "INTERNAL_ERROR", errorMessage)
		}
		return id, err
	}
	return id, nil
}

type holidayInput struct {
	Date string  `json:"tanggal"`
	Name string  `json:"nama"`
	Note *string `json:"keterangan"`
}

func (h *UATHandler) ListHolidays(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("tahun")
	if year == "" {
		year = time.Now().In(domain.Jakarta()).Format("2006")
	}
	rows, err := h.db.Query(r.Context(), `SELECT tanggal,nama,keterangan FROM holidays WHERE EXTRACT(YEAR FROM tanggal)=$1::int ORDER BY tanggal`, year)
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Kalender belum dapat dimuat")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var date time.Time
		var name string
		var note *string
		if rows.Scan(&date, &name, &note) != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Kalender belum dapat dimuat")
			return
		}
		items = append(items, map[string]any{"tanggal": date.Format("2006-01-02"), "nama": name, "keterangan": note})
	}
	response.Success(w, 200, items, "Kalender berhasil dimuat")
}
func (h *UATHandler) UpsertHolidays(w http.ResponseWriter, r *http.Request) {
	identity, ok := requireHR(w, r)
	if !ok {
		return
	}
	var input struct {
		Items []holidayInput `json:"items"`
	}
	if decodeJSON(r, &input) != nil || len(input.Items) == 0 || len(input.Items) > 366 {
		response.Error(w, 400, "INVALID_PARAM", "Daftar hari libur tidak valid")
		return
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Kalender belum dapat disimpan")
		return
	}
	defer tx.Rollback(r.Context())
	for _, item := range input.Items {
		if _, err = time.Parse("2006-01-02", item.Date); err != nil || strings.TrimSpace(item.Name) == "" {
			response.Error(w, 400, "INVALID_PARAM", "Tanggal dan nama hari libur wajib valid")
			return
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO holidays(tanggal,nama,keterangan,created_by) VALUES($1,$2,$3,$4) ON CONFLICT(tanggal) DO UPDATE SET nama=EXCLUDED.nama,keterangan=EXCLUDED.keterangan,updated_at=NOW()`, item.Date, strings.TrimSpace(item.Name), item.Note, identity.UserID); err != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Kalender belum dapat disimpan")
			return
		}
	}
	if tx.Commit(r.Context()) != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Kalender belum dapat disimpan")
		return
	}
	response.Success(w, 200, map[string]any{"updated": len(input.Items)}, "Kalender berhasil diperbarui")
}

type correctionInput struct {
	Date     string  `json:"tanggal"`
	CheckIn  *string `json:"waktu_check_in"`
	CheckOut *string `json:"waktu_check_out"`
	Reason   string  `json:"alasan"`
}

func (h *UATHandler) CreateCorrection(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok || identity.Role == domain.RoleTopManagement {
		response.Error(w, 403, "FORBIDDEN", "Anda tidak memiliki akses")
		return
	}
	var input correctionInput
	if decodeJSON(r, &input) != nil || strings.TrimSpace(input.Reason) == "" {
		response.Error(w, 400, "INVALID_PARAM", "Tanggal, waktu koreksi, dan alasan wajib diisi")
		return
	}
	if _, err := time.Parse("2006-01-02", input.Date); err != nil || (input.CheckIn == nil && input.CheckOut == nil) {
		response.Error(w, 400, "INVALID_PARAM", "Tanggal atau waktu koreksi tidak valid")
		return
	}
	status := "menunggu_hr"
	var hasSupervisor bool
	if err := h.db.QueryRow(r.Context(), `SELECT e.atasan_id IS NOT NULL FROM employees e WHERE e.id=$1`, identity.EmployeeID).Scan(&hasSupervisor); err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Koreksi belum dapat diajukan")
		return
	}
	if hasSupervisor {
		status = "menunggu_atasan"
	}
	var id uuid.UUID
	if err := h.db.QueryRow(r.Context(), `INSERT INTO attendance_corrections(user_id,tanggal,waktu_check_in,waktu_check_out,alasan,status) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, identity.UserID, input.Date, input.CheckIn, input.CheckOut, strings.TrimSpace(input.Reason), status).Scan(&id); err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Koreksi belum dapat diajukan")
		return
	}
	response.Success(w, 201, map[string]any{"id": id, "status": status}, "Koreksi absensi berhasil diajukan")
}

func (h *UATHandler) ListCorrections(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		return
	}
	query := `SELECT c.id,e.nama,TO_CHAR(c.tanggal,'YYYY-MM-DD'),TO_CHAR(c.waktu_check_in,'HH24:MI'),TO_CHAR(c.waktu_check_out,'HH24:MI'),c.alasan,c.status,c.created_at FROM attendance_corrections c JOIN users u ON u.id=c.user_id JOIN employees e ON e.id=u.employee_id WHERE `
	var rows interface {
		Close()
		Next() bool
		Scan(...any) error
	}
	var err error
	switch identity.Role {
	case domain.RoleEmployee:
		rows, err = h.db.Query(r.Context(), query+`c.user_id=$1 ORDER BY c.created_at DESC`, identity.UserID)
	case domain.RoleSupervisor:
		rows, err = h.db.Query(r.Context(), query+`e.atasan_id=$1 AND c.status='menunggu_atasan' ORDER BY c.created_at`, identity.EmployeeID)
	case domain.RoleHR:
		rows, err = h.db.Query(r.Context(), query+`c.status='menunggu_hr' OR c.user_id=$1 ORDER BY c.created_at DESC`, identity.UserID)
	default:
		response.Error(w, 403, "FORBIDDEN", "Anda tidak memiliki akses")
		return
	}
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Koreksi belum dapat dimuat")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, date, reason, status string
		var in, out *string
		var created time.Time
		if rows.Scan(&id, &name, &date, &in, &out, &reason, &status, &created) != nil {
			response.Error(w, 500, "INTERNAL_ERROR", "Koreksi belum dapat dimuat")
			return
		}
		items = append(items, map[string]any{"id": id, "nama_karyawan": name, "tanggal": date, "waktu_check_in": in, "waktu_check_out": out, "alasan": reason, "status": status, "created_at": created})
	}
	response.Success(w, 200, items, "Koreksi berhasil dimuat")
}

func (h *UATHandler) DecideCorrection(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.IdentityFromContext(r.Context())
	if !ok {
		return
	}
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		response.Error(w, 400, "INVALID_PARAM", "ID tidak valid")
		return
	}
	var input struct {
		Decision string  `json:"keputusan"`
		Note     *string `json:"catatan"`
	}
	if decodeJSON(r, &input) != nil || (input.Decision != "setujui" && input.Decision != "tolak") {
		response.Error(w, 400, "INVALID_PARAM", "Keputusan tidak valid")
		return
	}
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Keputusan belum tersimpan")
		return
	}
	defer tx.Rollback(r.Context())
	var requester uuid.UUID
	var employee, oldStatus, date string
	var checkIn, checkOut *string
	if err = tx.QueryRow(r.Context(), `SELECT c.user_id,e.id::text,c.status,TO_CHAR(c.tanggal,'YYYY-MM-DD'),TO_CHAR(c.waktu_check_in,'HH24:MI'),TO_CHAR(c.waktu_check_out,'HH24:MI') FROM attendance_corrections c JOIN users u ON u.id=c.user_id JOIN employees e ON e.id=u.employee_id WHERE c.id=$1 FOR UPDATE`, id).Scan(&requester, &employee, &oldStatus, &date, &checkIn, &checkOut); err != nil {
		response.Error(w, 404, "NOT_FOUND", "Koreksi tidak ditemukan")
		return
	}
	stage := ""
	next := ""
	if identity.Role == domain.RoleSupervisor && oldStatus == "menunggu_atasan" {
		var supervisor string
		if err = tx.QueryRow(r.Context(), `SELECT atasan_id::text FROM employees WHERE id=$1`, employee).Scan(&supervisor); err != nil || supervisor != identity.EmployeeID.String() {
			response.Error(w, 403, "FORBIDDEN", "Anda tidak memiliki akses")
			return
		}
		stage = "atasan"
		next = "menunggu_hr"
	} else if identity.Role == domain.RoleHR && oldStatus == "menunggu_hr" {
		stage = "hr"
		next = "disetujui"
	} else {
		response.Error(w, 403, "FORBIDDEN", "Anda tidak memiliki akses")
		return
	}
	if input.Decision == "tolak" {
		next = "ditolak"
	}
	if _, err = tx.Exec(r.Context(), `UPDATE attendance_corrections SET status=$2,updated_at=NOW() WHERE id=$1`, id, next); err != nil {
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO attendance_correction_approvals(correction_id,approver_id,tahap,keputusan,catatan) VALUES($1,$2,$3,$4,$5)`, id, identity.UserID, stage, map[bool]string{true: "approve", false: "reject"}[input.Decision == "setujui"], input.Note); err != nil {
		return
	}
	// Pada approval final, jam yang sudah ada diperbarui. Koreksi tidak membuat absensi
	// fiktif tanpa foto/lokasi. Input adalah jam Jakarta, lalu diubah menjadi instant UTC
	// sebelum disimpan ke kolom TIMESTAMPTZ.
	if next == "disetujui" {
		if checkIn != nil {
			var corrected time.Time
			corrected, err = correctedAttendanceTime(date, *checkIn)
			if err == nil {
				var tag pgconn.CommandTag
				tag, err = tx.Exec(r.Context(), `UPDATE attendances SET waktu_network=$3,waktu_local=$3,status=$4 WHERE user_id=$1 AND tanggal=$2 AND tipe='check_in'`, requester, date, corrected, domain.CheckInStatus(corrected))
				if err == nil && tag.RowsAffected() == 0 {
					err = errCorrectionAttendanceMissing
				}
			}
		}
		if err == nil && checkOut != nil {
			var corrected time.Time
			corrected, err = correctedAttendanceTime(date, *checkOut)
			if err == nil {
				var tag pgconn.CommandTag
				tag, err = tx.Exec(r.Context(), `UPDATE attendances SET waktu_network=$3,waktu_local=$3,status=$4 WHERE user_id=$1 AND tanggal=$2 AND tipe='check_out'`, requester, date, corrected, domain.CheckOutStatus(corrected))
				if err == nil && tag.RowsAffected() == 0 {
					err = errCorrectionAttendanceMissing
				}
			}
		}
	}
	if errors.Is(err, errCorrectionAttendanceMissing) {
		response.Error(w, 409, "ATTENDANCE_NOT_FOUND", "Absensi asal tidak ditemukan; koreksi tidak disetujui")
		return
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		response.Error(w, 500, "INTERNAL_ERROR", "Keputusan belum tersimpan")
		return
	}
	response.Success(w, 200, map[string]any{"id": id, "status": next}, "Keputusan koreksi tersimpan")
}

var errCorrectionAttendanceMissing = errors.New("attendance row for correction not found")

func correctedAttendanceTime(date, clock string) (time.Time, error) {
	local, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, domain.Jakarta())
	if err != nil {
		return time.Time{}, err
	}
	return local.UTC(), nil
}

// ParseBulkCSV validates a portable template; creation itself is handled by the employee service endpoint.
func ParseBulkCSV(reader io.Reader) ([][]string, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv has no rows")
	}
	return records, nil
}
