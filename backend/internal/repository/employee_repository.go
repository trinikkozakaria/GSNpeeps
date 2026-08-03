package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeRepository struct {
	pool *pgxpool.Pool
}

func (r *EmployeeRepository) ValidateCreate(
	ctx context.Context,
	command domain.CreateEmployee,
) error {
	exec := executor(ctx, r.pool)
	var organizationValid bool
	err := exec.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM positions p
			JOIN departments d ON d.id = p.department_id
			WHERE d.id = $1 AND p.id = $2
		)
	`, command.DepartmentID, command.PositionID).Scan(&organizationValid)
	if err != nil {
		return fmt.Errorf("validate employee organization: %w", err)
	}
	if !organizationValid {
		return ErrConflict
	}
	if command.SupervisorID != nil {
		var supervisorActive bool
		err := exec.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM employees
				WHERE id = $1 AND status = 'aktif' AND deleted_at IS NULL
			)
		`, *command.SupervisorID).Scan(&supervisorActive)
		if err != nil {
			return fmt.Errorf("validate employee supervisor: %w", err)
		}
		if !supervisorActive {
			return ErrNotFound
		}
	}
	return nil
}

func (r *EmployeeRepository) Create(
	ctx context.Context,
	command domain.CreateEmployee,
) (domain.EmployeeMutationResult, error) {
	exec := executor(ctx, r.pool)
	var employeeID uuid.UUID
	err := exec.QueryRow(ctx, `
		INSERT INTO employees (
			nip, nama, jenis_kelamin, tanggal_lahir, tanggal_join,
			department_id, position_id, atasan_id, status_pernikahan, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'aktif')
		RETURNING id
	`,
		command.NIP,
		command.Name,
		command.Gender,
		command.BirthDate,
		command.JoinDate,
		command.DepartmentID,
		command.PositionID,
		command.SupervisorID,
		command.MaritalStatus,
	).Scan(&employeeID)
	if err != nil {
		return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
	}

	_, err = exec.Exec(ctx, `
		INSERT INTO employee_addresses (
			employee_id, jalan, kelurahan, kecamatan, kota, provinsi
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		employeeID,
		command.Address.Street,
		command.Address.Village,
		command.Address.District,
		command.Address.City,
		command.Address.Province,
	)
	if err != nil {
		return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
	}
	_, err = exec.Exec(ctx, `
		INSERT INTO employee_ktp (employee_id, nomor_ktp)
		VALUES ($1, $2)
	`, employeeID, command.KTPNumber)
	if err != nil {
		return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
	}
	_, err = exec.Exec(ctx, `
		INSERT INTO employee_contracts (
			employee_id, nomor_kontrak, tanggal_mulai, tanggal_berakhir,
			jenis_kontrak, status
		)
		VALUES ($1, $2, $3, $4, $5, 'aktif')
	`,
		employeeID,
		command.Contract.Number,
		command.Contract.StartDate,
		command.Contract.EndDate,
		command.Contract.Type,
	)
	if err != nil {
		return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
	}

	var userID uuid.UUID
	err = exec.QueryRow(ctx, `
		INSERT INTO users (
			employee_id, email, password_hash, role_id, account_locked
		)
		VALUES (
			$1, $2, $3, (SELECT id FROM roles WHERE nama = $4), TRUE
		)
		RETURNING id
	`, employeeID, command.Email, command.PasswordHash, command.Role).Scan(&userID)
	if err != nil {
		return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
	}
	return domain.EmployeeMutationResult{EmployeeID: employeeID, UserID: &userID}, nil
}

func (r *EmployeeRepository) ValidateMutation(
	ctx context.Context,
	employeeID uuid.UUID,
	changes domain.EmployeeChanges,
) error {
	exec := executor(ctx, r.pool)
	if changes.DepartmentID != nil {
		var exists bool
		if err := exec.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`,
			*changes.DepartmentID).Scan(&exists); err != nil {
			return fmt.Errorf("validate employee department: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
	}
	if changes.PositionID != nil || changes.DepartmentID != nil {
		var valid bool
		err := exec.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM employees e
				JOIN positions p ON p.id = COALESCE($1::uuid, e.position_id)
				WHERE e.id = $3
				  AND p.department_id = COALESCE($2::uuid, e.department_id)
			)
		`, changes.PositionID, changes.DepartmentID, employeeID).Scan(&valid)
		if err != nil {
			return fmt.Errorf("validate employee position: %w", err)
		}
		if !valid {
			return ErrConflict
		}
	}
	if changes.SupervisorSet && changes.SupervisorID != nil {
		if *changes.SupervisorID == employeeID {
			return ErrConflict
		}
		var active bool
		err := exec.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM employees
				WHERE id = $1 AND status = 'aktif' AND deleted_at IS NULL
			)
		`, *changes.SupervisorID).Scan(&active)
		if err != nil {
			return fmt.Errorf("validate employee supervisor: %w", err)
		}
		if !active {
			return ErrNotFound
		}
	}
	return nil
}

func (r *EmployeeRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	changes domain.EmployeeChanges,
) (domain.EmployeeMutationResult, error) {
	exec := executor(ctx, r.pool)
	var userID *uuid.UUID
	if err := exec.QueryRow(ctx, `
		SELECT u.id
		FROM employees e
		LEFT JOIN users u ON u.employee_id = e.id
		WHERE e.id = $1 AND e.deleted_at IS NULL
		FOR UPDATE OF e
	`, id).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return domain.EmployeeMutationResult{}, ErrNotFound
	} else if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("lock employee for update: %w", err)
	}

	setParts := make([]string, 0, 10)
	args := []any{id}
	add := func(column string, value any) {
		args = append(args, value)
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if changes.Name != nil {
		add("nama", *changes.Name)
	}
	if changes.Gender != nil {
		add("jenis_kelamin", *changes.Gender)
	}
	if changes.BirthDate != nil {
		add("tanggal_lahir", *changes.BirthDate)
	}
	if changes.JoinDate != nil {
		add("tanggal_join", *changes.JoinDate)
	}
	if changes.DepartmentID != nil {
		add("department_id", *changes.DepartmentID)
	}
	if changes.PositionID != nil {
		add("position_id", *changes.PositionID)
	}
	if changes.SupervisorSet {
		add("atasan_id", changes.SupervisorID)
	}
	if changes.MaritalStatus != nil {
		add("status_pernikahan", *changes.MaritalStatus)
	}
	if changes.Status != nil {
		add("status", *changes.Status)
		if *changes.Status == "nonaktif" {
			setParts = append(setParts, "deleted_at = NOW()")
		}
	}
	if len(setParts) > 0 {
		setParts = append(setParts, "updated_at = NOW()")
		if _, err := exec.Exec(ctx, `
			UPDATE employees SET `+strings.Join(setParts, ", ")+` WHERE id = $1
		`, args...); err != nil {
			return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
		}
	}
	if changes.Email != nil || changes.Role != nil {
		if userID == nil {
			return domain.EmployeeMutationResult{}, ErrConflict
		}
		userSet := make([]string, 0, 3)
		userArgs := []any{*userID}
		if changes.Email != nil {
			userArgs = append(userArgs, *changes.Email)
			userSet = append(userSet, fmt.Sprintf("email = $%d", len(userArgs)))
		}
		if changes.Role != nil {
			userArgs = append(userArgs, string(*changes.Role))
			userSet = append(userSet, fmt.Sprintf(
				"role_id = (SELECT id FROM roles WHERE nama = $%d)",
				len(userArgs),
			))
		}
		userSet = append(userSet, "updated_at = NOW()")
		if _, err := exec.Exec(ctx, `
			UPDATE users SET `+strings.Join(userSet, ", ")+` WHERE id = $1
		`, userArgs...); err != nil {
			return domain.EmployeeMutationResult{}, mapEmployeeMutationError(err)
		}
	}
	return domain.EmployeeMutationResult{EmployeeID: id, UserID: userID}, nil
}

func (r *EmployeeRepository) SoftDelete(
	ctx context.Context,
	id uuid.UUID,
) (domain.EmployeeMutationResult, error) {
	exec := executor(ctx, r.pool)
	var userID *uuid.UUID
	err := exec.QueryRow(ctx, `
		WITH updated AS (
			UPDATE employees
			SET status = 'nonaktif', deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id
		)
		SELECT updated.id, u.id
		FROM updated
		LEFT JOIN users u ON u.employee_id = updated.id
	`, id).Scan(&id, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.EmployeeMutationResult{}, ErrNotFound
	}
	if err != nil {
		return domain.EmployeeMutationResult{}, fmt.Errorf("soft delete employee: %w", err)
	}
	return domain.EmployeeMutationResult{EmployeeID: id, UserID: userID}, nil
}

func mapEmployeeMutationError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505", "23503", "23514":
			return ErrConflict
		}
	}
	return fmt.Errorf("mutate employee: %w", err)
}

func NewEmployeeRepository(pool *pgxpool.Pool) *EmployeeRepository {
	return &EmployeeRepository{pool: pool}
}

func (r *EmployeeRepository) ListDepartments(ctx context.Context) ([]domain.Department, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, nama FROM departments ORDER BY nama, id`)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Department, 0)
	for rows.Next() {
		var item domain.Department
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *EmployeeRepository) ListPositions(
	ctx context.Context,
	departmentID *uuid.UUID,
) ([]domain.Position, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, department_id, nama
		FROM positions
		WHERE ($1::uuid IS NULL OR department_id = $1)
		ORDER BY nama, id
	`, departmentID)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Position, 0)
	for rows.Next() {
		var item domain.Position
		if err := rows.Scan(&item.ID, &item.DepartmentID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *EmployeeRepository) List(
	ctx context.Context,
	filter domain.EmployeeFilter,
) (domain.EmployeePage, error) {
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		search = "%" + search + "%"
	}
	offset := (filter.Page - 1) * filter.Limit
	args := []any{search, filter.DepartmentID, filter.Status}
	// Pencarian dibatasi pada nama dan NIP sesuai API Contract v1.1 §5.1.
	where := `
		($1 = '' OR e.nama ILIKE $1 OR e.nip ILIKE $1)
		AND ($2::uuid IS NULL OR e.department_id = $2)
		AND ($3 = '' OR e.status = $3)`

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM employees e
		LEFT JOIN users u ON u.employee_id = e.id
		WHERE `+where, args...).Scan(&total); err != nil {
		return domain.EmployeePage{}, fmt.Errorf("count employees: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.nip, e.nama, COALESCE(u.email, ''),
		       COALESCE(d.nama, ''), COALESCE(p.nama, ''), e.status
		FROM employees e
		LEFT JOIN users u ON u.employee_id = e.id
		LEFT JOIN departments d ON d.id = e.department_id
		LEFT JOIN positions p ON p.id = e.position_id
		WHERE `+where+`
		ORDER BY e.created_at DESC, e.id
		LIMIT $4 OFFSET $5
	`, append(args, filter.Limit, offset)...)
	if err != nil {
		return domain.EmployeePage{}, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EmployeeSummary, 0)
	for rows.Next() {
		var item domain.EmployeeSummary
		if err := rows.Scan(
			&item.ID,
			&item.NIP,
			&item.Name,
			&item.Email,
			&item.Department,
			&item.Position,
			&item.Status,
		); err != nil {
			return domain.EmployeePage{}, fmt.Errorf("scan employee summary: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.EmployeePage{}, fmt.Errorf("iterate employees: %w", err)
	}
	return domain.EmployeePage{Items: items, Total: total, Page: filter.Page, Limit: filter.Limit}, nil
}

// FindByID mengambil detail Point 1-11. `salaryPeriod` membatasi gaji ke bulan berjalan
// sesuai PRD; histori gaji penuh tidak pernah dibaca di sini.
func (r *EmployeeRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
	salaryPeriod string,
) (domain.EmployeeDetail, error) {
	var item domain.EmployeeDetail
	err := r.pool.QueryRow(ctx, `
		SELECT e.id, e.nip, e.nama, COALESCE(u.email, ''),
		       COALESCE(d.nama, ''), COALESCE(p.nama, ''), e.status,
		       e.jenis_kelamin, TO_CHAR(e.tanggal_lahir, 'YYYY-MM-DD'),
		       TO_CHAR(e.tanggal_join, 'YYYY-MM-DD'),
		       e.department_id, e.position_id, e.atasan_id, e.status_pernikahan
		FROM employees e
		LEFT JOIN users u ON u.employee_id = e.id
		LEFT JOIN departments d ON d.id = e.department_id
		LEFT JOIN positions p ON p.id = e.position_id
		WHERE e.id = $1
	`, id).Scan(
		&item.ID,
		&item.NIP,
		&item.Name,
		&item.Email,
		&item.Department,
		&item.Position,
		&item.Status,
		&item.Gender,
		&item.BirthDate,
		&item.JoinDate,
		&item.DepartmentID,
		&item.PositionID,
		&item.SupervisorID,
		&item.MaritalStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.EmployeeDetail{}, ErrNotFound
	}
	if err != nil {
		return domain.EmployeeDetail{}, fmt.Errorf("find employee: %w", err)
	}
	if err := r.loadEmployeeDetails(ctx, &item, salaryPeriod); err != nil {
		return domain.EmployeeDetail{}, err
	}
	return item, nil
}

func (r *EmployeeRepository) loadEmployeeDetails(
	ctx context.Context,
	item *domain.EmployeeDetail,
	salaryPeriod string,
) error {
	var address domain.EmployeeAddress
	err := r.pool.QueryRow(ctx, `
		SELECT jalan, kelurahan, kecamatan, kota, provinsi
		FROM employee_addresses WHERE employee_id = $1
	`, item.ID).Scan(
		&address.Street,
		&address.Village,
		&address.District,
		&address.City,
		&address.Province,
	)
	if err == nil {
		item.Address = &address
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load employee address: %w", err)
	}

	var ktp domain.EmployeeKTP
	err = r.pool.QueryRow(ctx, `
		SELECT nomor_ktp, file_url FROM employee_ktp WHERE employee_id = $1
	`, item.ID).Scan(&ktp.Number, &ktp.FileURL)
	if err == nil {
		item.KTP = &ktp
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load employee ktp: %w", err)
	}

	var npwp domain.EmployeeNPWP
	var npwpNumber *string
	err = r.pool.QueryRow(ctx, `
		SELECT nomor_npwp, file_url FROM employee_npwp WHERE employee_id = $1
	`, item.ID).Scan(&npwpNumber, &npwp.FileURL)
	if err == nil && npwpNumber != nil {
		npwp.Number = *npwpNumber
		item.NPWP = &npwp
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load employee npwp: %w", err)
	}

	if err := r.loadEmployeeContracts(ctx, item); err != nil {
		return err
	}
	if err := r.loadEmployeeBPJS(ctx, item); err != nil {
		return err
	}
	if err := r.loadEmergencyContacts(ctx, item); err != nil {
		return err
	}
	if err := r.loadEmployeeEducation(ctx, item); err != nil {
		return err
	}
	if err := r.loadPositionHistory(ctx, item); err != nil {
		return err
	}
	return r.loadCurrentSalary(ctx, item, salaryPeriod)
}

func (r *EmployeeRepository) loadEmployeeContracts(
	ctx context.Context,
	item *domain.EmployeeDetail,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT nomor_kontrak, TO_CHAR(tanggal_mulai, 'YYYY-MM-DD'),
		       TO_CHAR(tanggal_berakhir, 'YYYY-MM-DD'), jenis_kontrak, file_url, status
		FROM employee_contracts
		WHERE employee_id = $1
		ORDER BY tanggal_mulai DESC, id
	`, item.ID)
	if err != nil {
		return fmt.Errorf("load employee contracts: %w", err)
	}
	defer rows.Close()
	item.Contracts = make([]domain.EmployeeContract, 0)
	for rows.Next() {
		var contract domain.EmployeeContract
		if err := rows.Scan(
			&contract.Number,
			&contract.StartDate,
			&contract.EndDate,
			&contract.Type,
			&contract.FileURL,
			&contract.Status,
		); err != nil {
			return fmt.Errorf("scan employee contract: %w", err)
		}
		item.Contracts = append(item.Contracts, contract)
	}
	return rows.Err()
}

// loadEmployeeBPJS memetakan satu baris `employee_bpjs` menjadi koleksi per jenis
// kepesertaan sesuai schema EmployeeBPJS; nomor kosong tidak dikirim.
func (r *EmployeeRepository) loadEmployeeBPJS(
	ctx context.Context,
	item *domain.EmployeeDetail,
) error {
	item.BPJS = make([]domain.EmployeeBPJS, 0, 2)
	var health, employment *string
	err := r.pool.QueryRow(ctx, `
		SELECT no_bpjs_kesehatan, no_bpjs_ketenagakerjaan
		FROM employee_bpjs WHERE employee_id = $1
	`, item.ID).Scan(&health, &employment)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load employee bpjs: %w", err)
	}
	if health != nil && *health != "" {
		item.BPJS = append(item.BPJS, domain.EmployeeBPJS{Type: "kesehatan", Number: *health})
	}
	if employment != nil && *employment != "" {
		item.BPJS = append(item.BPJS, domain.EmployeeBPJS{Type: "ketenagakerjaan", Number: *employment})
	}
	return nil
}

func (r *EmployeeRepository) loadEmergencyContacts(
	ctx context.Context,
	item *domain.EmployeeDetail,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT nama, hubungan, no_hp
		FROM employee_emergency_contacts
		WHERE employee_id = $1
		ORDER BY nama, id
	`, item.ID)
	if err != nil {
		return fmt.Errorf("load employee emergency contacts: %w", err)
	}
	defer rows.Close()
	item.EmergencyContacts = make([]domain.EmergencyContact, 0)
	for rows.Next() {
		var contact domain.EmergencyContact
		if err := rows.Scan(&contact.Name, &contact.Relationship, &contact.Phone); err != nil {
			return fmt.Errorf("scan employee emergency contact: %w", err)
		}
		item.EmergencyContacts = append(item.EmergencyContacts, contact)
	}
	return rows.Err()
}

func (r *EmployeeRepository) loadEmployeeEducation(
	ctx context.Context,
	item *domain.EmployeeDetail,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT jenjang, institusi, tahun_lulus
		FROM employee_education
		WHERE employee_id = $1
		ORDER BY tahun_lulus DESC NULLS LAST, id
	`, item.ID)
	if err != nil {
		return fmt.Errorf("load employee education: %w", err)
	}
	defer rows.Close()
	item.Education = make([]domain.EducationHistory, 0)
	for rows.Next() {
		var education domain.EducationHistory
		var year *int16
		if err := rows.Scan(&education.Level, &education.Institution, &year); err != nil {
			return fmt.Errorf("scan employee education: %w", err)
		}
		if year != nil {
			value := int(*year)
			education.GraduationYear = &value
		}
		item.Education = append(item.Education, education)
	}
	return rows.Err()
}

func (r *EmployeeRepository) loadPositionHistory(
	ctx context.Context,
	item *domain.EmployeeDetail,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT h.department_id, d.nama, h.position_id, p.department_id, p.nama,
		       TO_CHAR(h.tanggal_mulai, 'YYYY-MM-DD'),
		       TO_CHAR(h.tanggal_selesai, 'YYYY-MM-DD')
		FROM employee_position_history h
		LEFT JOIN departments d ON d.id = h.department_id
		LEFT JOIN positions p ON p.id = h.position_id
		WHERE h.employee_id = $1
		ORDER BY h.tanggal_mulai DESC, h.id
	`, item.ID)
	if err != nil {
		return fmt.Errorf("load employee position history: %w", err)
	}
	defer rows.Close()
	item.PositionHistory = make([]domain.PositionHistory, 0)
	for rows.Next() {
		var (
			entry              domain.PositionHistory
			departmentID       *uuid.UUID
			departmentName     *string
			positionID         *uuid.UUID
			positionDepartment *uuid.UUID
			positionName       *string
		)
		if err := rows.Scan(
			&departmentID,
			&departmentName,
			&positionID,
			&positionDepartment,
			&positionName,
			&entry.StartDate,
			&entry.EndDate,
		); err != nil {
			return fmt.Errorf("scan employee position history: %w", err)
		}
		if departmentID != nil && departmentName != nil {
			entry.Department = &domain.Department{ID: *departmentID, Name: *departmentName}
		}
		if positionID != nil && positionDepartment != nil && positionName != nil {
			entry.Position = &domain.Position{
				ID:           *positionID,
				DepartmentID: *positionDepartment,
				Name:         *positionName,
			}
		}
		item.PositionHistory = append(item.PositionHistory, entry)
	}
	return rows.Err()
}

func (r *EmployeeRepository) loadCurrentSalary(
	ctx context.Context,
	item *domain.EmployeeDetail,
	period string,
) error {
	if period == "" {
		return nil
	}
	var salary domain.CurrentSalary
	err := r.pool.QueryRow(ctx, `
		SELECT periode, gaji_pokok::float8, tunjangan::float8, potongan::float8,
		       take_home_pay::float8
		FROM employee_salaries
		WHERE employee_id = $1 AND periode = $2
	`, item.ID, period).Scan(
		&salary.Period,
		&salary.BasePay,
		&salary.Allowance,
		&salary.Deduction,
		&salary.TakeHomePay,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load employee current salary: %w", err)
	}
	item.CurrentSalary = &salary
	return nil
}

// FindDocuments mengembalikan metadata dokumen; isi berkas tidak pernah dibaca dari database.
func (r *EmployeeRepository) FindDocuments(
	ctx context.Context,
	employeeID uuid.UUID,
) ([]domain.EmployeeDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, jenis_dokumen, nama_file, file_url, uploaded_at
		FROM employee_documents
		WHERE employee_id = $1
		ORDER BY uploaded_at DESC, id
	`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list employee documents: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EmployeeDocument, 0)
	for rows.Next() {
		var item domain.EmployeeDocument
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.FileName,
			&item.FileURL,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan employee document: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CreateDocument menyimpan locator dokumen. Dipanggil di dalam transaction agar kegagalan
// setelah upload dapat dikompensasi oleh service.
func (r *EmployeeRepository) CreateDocument(
	ctx context.Context,
	document domain.NewEmployeeDocument,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := executor(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO employee_documents (employee_id, jenis_dokumen, nama_file, file_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, document.EmployeeID, document.Type, document.FileName, document.FileURL).Scan(&id)
	if err != nil {
		return uuid.Nil, mapEmployeeMutationError(err)
	}
	return id, nil
}

// ExistsActive memastikan employee ada dan belum soft-deleted sebelum operasi dokumen.
func (r *EmployeeRepository) ExistsActive(ctx context.Context, employeeID uuid.UUID) error {
	var exists bool
	err := executor(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND deleted_at IS NULL)
	`, employeeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

// ExportRows membaca dataset export dengan filter yang sama seperti list. Batas maksimum
// menjaga export tetap terikat dan tidak menarik seluruh tabel tanpa kendali.
func (r *EmployeeRepository) ExportRows(
	ctx context.Context,
	query domain.EmployeeExportQuery,
	maxRows int,
) ([]domain.EmployeeSummary, error) {
	search := strings.TrimSpace(query.Filter.Search)
	if search != "" {
		search = "%" + search + "%"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.nip, e.nama, COALESCE(u.email, ''),
		       COALESCE(d.nama, ''), COALESCE(p.nama, ''), e.status
		FROM employees e
		LEFT JOIN users u ON u.employee_id = e.id
		LEFT JOIN departments d ON d.id = e.department_id
		LEFT JOIN positions p ON p.id = e.position_id
		WHERE ($1::uuid IS NULL OR e.id = $1)
		  AND ($2 = '' OR e.nama ILIKE $2 OR e.nip ILIKE $2)
		  AND ($3::uuid IS NULL OR e.department_id = $3)
		  AND ($4 = '' OR e.status = $4)
		ORDER BY e.nama, e.id
		LIMIT $5
	`, query.EmployeeID, search, query.Filter.DepartmentID, query.Filter.Status, maxRows)
	if err != nil {
		return nil, fmt.Errorf("query employee export rows: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EmployeeSummary, 0)
	for rows.Next() {
		var item domain.EmployeeSummary
		if err := rows.Scan(
			&item.ID,
			&item.NIP,
			&item.Name,
			&item.Email,
			&item.Department,
			&item.Position,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan employee export row: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
