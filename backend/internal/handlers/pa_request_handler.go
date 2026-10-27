package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/models"
	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PARequestHandler struct {
	DB *pgxpool.Pool
}

func NewPARequestHandler(db *pgxpool.Pool) *PARequestHandler {
	return &PARequestHandler{DB: db}
}

func (h *PARequestHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")
	role := c.GetString("role")

	page, err := parsePositiveInt(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page harus berupa bilangan positif."})
		return
	}

	limit, err := parsePositiveInt(c.DefaultQuery("limit", "10"))
	if err != nil || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit harus antara 1 dan 100."})
		return
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" && !validRequestStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filter status tidak valid."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	where := []string{"deleted_at IS NULL"}
	args := []any{}
	argPos := 1

	// Staff hanya dapat melihat request miliknya.
	if role == "staff" {
		where = append(where, "created_by = $"+strconv.Itoa(argPos))
		args = append(args, userID)
		argPos++
	}

	if status != "" {
		where = append(where, "status = $"+strconv.Itoa(argPos))
		args = append(args, status)
		argPos++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQuery := "SELECT COUNT(*) FROM pa_requests WHERE " + whereSQL
	if err := h.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar request."})
		return
	}

	offset := (page - 1) * limit
	listArgs := append(append([]any{}, args...), limit, offset)

	query := `
		SELECT id, request_number, created_by, payer_name, service_type,
		       provider_npi, patient_reference, diagnosis, status,
		       checklist_version_id, payer_reference, submitted_at, payer_decision,
		       payer_decision_at, payer_decision_note, created_at, updated_at
		FROM pa_requests
		WHERE ` + whereSQL + `
		ORDER BY created_at DESC, id DESC
		LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)

	rows, err := h.DB.Query(ctx, query, listArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar request."})
		return
	}
	defer rows.Close()

	items := make([]models.PARequest, 0)
	for rows.Next() {
		item, scanErr := scanPARequest(rows)
		if scanErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca data request."})
			return
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca daftar request."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *PARequestHandler) Create(c *gin.Context) {
	var input models.CreatePARequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format atau field request tidak valid."})
		return
	}

	normalizeInput(
		&input.PayerName,
		&input.ServiceType,
		&input.ProviderNPI,
		&input.PatientReference,
		&input.Diagnosis,
	)
	if !validInput(input.PayerName, input.ServiceType, input.ProviderNPI, input.PatientReference, input.Diagnosis) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field wajib tidak boleh kosong; NPI harus terdiri dari 10 digit jika diisi."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	requestNumber, err := services.GenerateRequestNumber(time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat nomor request."})
		return
	}

	userID := c.GetInt64("user_id")
	checklistVersionID, err := findActiveChecklistVersion(ctx, h.DB, input.PayerName, input.ServiceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist aktif gagal dicari."})
		return
	}
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaksi pembuatan permohonan gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item models.PARequest
	err = tx.QueryRow(ctx, `
		INSERT INTO pa_requests (
			request_number, created_by, payer_name, service_type,
			provider_npi, patient_reference, diagnosis, status,
			checklist_version_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft', $8)
		RETURNING id, request_number, created_by, payer_name, service_type,
		          provider_npi, patient_reference, diagnosis, status,
		          checklist_version_id, payer_reference, submitted_at, payer_decision,
		          payer_decision_at, payer_decision_note, created_at, updated_at
	`,
		requestNumber, userID, input.PayerName, input.ServiceType,
		input.ProviderNPI, input.PatientReference, input.Diagnosis,
		checklistVersionID,
	).Scan(
		&item.ID, &item.RequestNumber, &item.CreatedBy, &item.PayerName,
		&item.ServiceType, &item.ProviderNPI, &item.PatientReference,
		&item.Diagnosis, &item.Status, &item.ChecklistVersionID,
		&item.PayerReference, &item.SubmittedAt, &item.PayerDecision,
		&item.PayerDecisionAt, &item.PayerDecisionNote,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Request tidak dapat dibuat. Periksa checklist version dan data yang dikirim."})
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO pa_request_status_history (pa_request_id,actor_id,from_status,to_status,action) VALUES ($1,$2,NULL,'draft','created')`, item.ID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat awal permohonan gagal disimpan."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Permohonan gagal disimpan."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *PARequestHandler) GetByID(c *gin.Context) {
	id, ok := parseRequestID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, err := h.findByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail request."})
		return
	}

	if !canReadRequest(c, item) {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PARequestHandler) Update(c *gin.Context) {
	id, ok := parseRequestID(c)
	if !ok {
		return
	}

	var input models.UpdatePARequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format atau field request tidak valid."})
		return
	}

	normalizeInput(
		&input.PayerName,
		&input.ServiceType,
		&input.ProviderNPI,
		&input.PatientReference,
		&input.Diagnosis,
	)
	if !validInput(input.PayerName, input.ServiceType, input.ProviderNPI, input.PatientReference, input.Diagnosis) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field wajib tidak boleh kosong; NPI harus terdiri dari 10 digit jika diisi."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	existing, err := h.findByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil request."})
		return
	}

	if !canModifyRequest(c, existing) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin mengubah request ini."})
		return
	}
	if existing.Status != "draft" && existing.Status != "needs_information" {
		c.JSON(http.StatusConflict, gin.H{"error": "Permohonan hanya dapat diubah saat draft atau setelah dikembalikan Reviewer."})
		return
	}
	checklistVersionID := existing.ChecklistVersionID
	if !strings.EqualFold(existing.PayerName, input.PayerName) || !strings.EqualFold(existing.ServiceType, input.ServiceType) || checklistVersionID == nil {
		checklistVersionID, err = findActiveChecklistVersion(ctx, h.DB, input.PayerName, input.ServiceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist aktif gagal dicari."})
			return
		}
	}

	var item models.PARequest
	err = h.DB.QueryRow(ctx, `
		UPDATE pa_requests
		SET payer_name = $1,
		    service_type = $2,
		    provider_npi = $3,
		    patient_reference = $4,
		    diagnosis = $5,
		    checklist_version_id = $6,
		    updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL AND status IN ('draft', 'needs_information')
		RETURNING id, request_number, created_by, payer_name, service_type,
		          provider_npi, patient_reference, diagnosis, status,
		          checklist_version_id, payer_reference, submitted_at, payer_decision,
		          payer_decision_at, payer_decision_note, created_at, updated_at
	`,
		input.PayerName, input.ServiceType, input.ProviderNPI,
		input.PatientReference, input.Diagnosis, checklistVersionID, id,
	).Scan(
		&item.ID, &item.RequestNumber, &item.CreatedBy, &item.PayerName,
		&item.ServiceType, &item.ProviderNPI, &item.PatientReference,
		&item.Diagnosis, &item.Status, &item.ChecklistVersionID,
		&item.PayerReference, &item.SubmittedAt, &item.PayerDecision,
		&item.PayerDecisionAt, &item.PayerDecisionNote,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Request tidak dapat diperbarui. Periksa checklist version dan status request."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PARequestHandler) Delete(c *gin.Context) {
	id, ok := parseRequestID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	existing, err := h.findByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil request."})
		return
	}

	if !canModifyRequest(c, existing) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin menghapus request ini."})
		return
	}
	if existing.Status != "draft" {
		c.JSON(http.StatusConflict, gin.H{"error": "Hanya request berstatus draft yang dapat dihapus."})
		return
	}

	tag, err := h.DB.Exec(ctx, `
		UPDATE pa_requests
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL AND status = 'draft'
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus request."})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Request sudah dihapus atau tidak lagi berstatus draft."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PA request berhasil dihapus."})
}

func (h *PARequestHandler) findByID(ctx context.Context, id int64) (models.PARequest, error) {
	row := h.DB.QueryRow(ctx, `
		SELECT id, request_number, created_by, payer_name, service_type,
		       provider_npi, patient_reference, diagnosis, status,
		       checklist_version_id, payer_reference, submitted_at, payer_decision,
		       payer_decision_at, payer_decision_note, created_at, updated_at
		FROM pa_requests
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return scanPARequest(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPARequest(row rowScanner) (models.PARequest, error) {
	var item models.PARequest
	var checklistID pgtype.Int8

	err := row.Scan(
		&item.ID,
		&item.RequestNumber,
		&item.CreatedBy,
		&item.PayerName,
		&item.ServiceType,
		&item.ProviderNPI,
		&item.PatientReference,
		&item.Diagnosis,
		&item.Status,
		&checklistID,
		&item.PayerReference,
		&item.SubmittedAt,
		&item.PayerDecision,
		&item.PayerDecisionAt,
		&item.PayerDecisionNote,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}

	if checklistID.Valid {
		item.ChecklistVersionID = &checklistID.Int64
	}

	return item, nil
}

func parsePositiveInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("value must be a positive integer")
	}
	return value, nil
}

func parseRequestID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID request tidak valid."})
		return 0, false
	}
	return id, true
}

func validRequestStatus(status string) bool {
	switch status {
	case "draft", "ready_for_review", "in_review", "needs_information", "ready_for_submission", "submitted", "completed", "cancelled", "processing", "error":
		return true
	default:
		return false
	}
}

func normalizeInput(payer, service, npi, patient, diagnosis *string) {
	*payer = strings.TrimSpace(*payer)
	*service = strings.TrimSpace(*service)
	*npi = strings.TrimSpace(*npi)
	*patient = strings.TrimSpace(*patient)
	*diagnosis = strings.TrimSpace(*diagnosis)
}

func validInput(payer, service, npi, patient, diagnosis string) bool {
	if payer == "" || service == "" || patient == "" || diagnosis == "" {
		return false
	}
	if npi != "" {
		if len(npi) != 10 {
			return false
		}
		for _, r := range npi {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func canReadRequest(c *gin.Context, item models.PARequest) bool {
	role := c.GetString("role")
	return role == "admin" || role == "reviewer" ||
		(role == "staff" && c.GetInt64("user_id") == item.CreatedBy)
}

func canModifyRequest(c *gin.Context, item models.PARequest) bool {
	role := c.GetString("role")
	return role == "staff" && c.GetInt64("user_id") == item.CreatedBy
}
