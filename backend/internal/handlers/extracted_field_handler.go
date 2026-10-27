package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExtractedFieldHandler struct {
	DB *pgxpool.Pool
}

func NewExtractedFieldHandler(db *pgxpool.Pool) *ExtractedFieldHandler {
	return &ExtractedFieldHandler{DB: db}
}

type extractedFieldResponse struct {
	ID                 int64      `json:"id"`
	PARequestID        int64      `json:"pa_request_id"`
	DocumentID         *int64     `json:"document_id"`
	FieldKey           string     `json:"field_key"`
	FieldValue         *string    `json:"field_value"`
	Confidence         *float64   `json:"confidence"`
	EvidenceText       *string    `json:"evidence_text"`
	VerificationStatus string     `json:"verification_status"`
	ReviewedBy         *int64     `json:"reviewed_by"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

type updateExtractedFieldRequest struct {
	FieldValue         *string `json:"field_value"`
	VerificationStatus *string `json:"verification_status"`
}

func (h *ExtractedFieldHandler) List(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}

	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	allowed, err := h.canAccessRequest(ctx, c, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa akses PA request."})
		return
	}
	if !allowed {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}

	rows, err := h.DB.Query(ctx, `
SELECT
id, pa_request_id, document_id, field_key, field_value,
confidence, evidence_text, verification_status,
reviewed_by, reviewed_at, created_at
FROM extracted_fields
WHERE pa_request_id = $1
ORDER BY id
`, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil extracted fields."})
		return
	}
	defer rows.Close()

	fields := make([]extractedFieldResponse, 0)
	for rows.Next() {
		var field extractedFieldResponse
		if err := rows.Scan(
			&field.ID,
			&field.PARequestID,
			&field.DocumentID,
			&field.FieldKey,
			&field.FieldValue,
			&field.Confidence,
			&field.EvidenceText,
			&field.VerificationStatus,
			&field.ReviewedBy,
			&field.ReviewedAt,
			&field.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca extracted fields."})
			return
		}
		fields = append(fields, field)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan pembacaan extracted fields."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"pa_request_id": requestID,
			"fields":        fields,
			"count":         len(fields),
		},
	})
}

func (h *ExtractedFieldHandler) Update(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}

	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	fieldID, err := strconv.ParseInt(c.Param("fieldId"), 10, 64)
	if err != nil || fieldID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID extracted field tidak valid."})
		return
	}

	var input updateExtractedFieldRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Body JSON tidak valid."})
		return
	}

	if input.FieldValue == nil && input.VerificationStatus == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Isi field_value atau verification_status yang ingin diperbarui.",
		})
		return
	}

	if input.VerificationStatus != nil {
		status := strings.TrimSpace(*input.VerificationStatus)
		switch status {
		case "pending", "verified", "corrected", "uncertain":
			input.VerificationStatus = &status
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "verification_status harus pending, verified, corrected, atau uncertain.",
			})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var createdBy int64
	var requestStatus string
	err = h.DB.QueryRow(ctx, `
SELECT pr.created_by, pr.status
FROM extracted_fields ef
JOIN pa_requests pr ON pr.id = ef.pa_request_id
WHERE ef.id = $1
  AND ef.pa_request_id = $2
  AND pr.deleted_at IS NULL
`, fieldID, requestID).Scan(&createdBy, &requestStatus)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extracted field tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa extracted field."})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt64("user_id")
	allowed := (role == "staff" && userID == createdBy && input.VerificationStatus == nil &&
		(requestStatus == "draft" || requestStatus == "needs_information" || requestStatus == "payer_needs_information")) ||
		(role == "reviewer" && requestStatus == "in_review")

	if !allowed {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extracted field tidak ditemukan."})
		return
	}

	tag, err := h.DB.Exec(ctx, `
UPDATE extracted_fields
SET field_value = COALESCE($1::text, field_value),
    verification_status = COALESCE($2::varchar, verification_status),
    reviewed_by = CASE WHEN $2::varchar IS NULL THEN reviewed_by ELSE $3 END,
    reviewed_at = CASE WHEN $2::varchar IS NULL THEN reviewed_at ELSE NOW() END
WHERE id = $4
  AND pa_request_id = $5
`, input.FieldValue, input.VerificationStatus, userID, fieldID, requestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui extracted field."})
		return
	}
	if tag.RowsAffected() != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extracted field tidak ditemukan."})
		return
	}

	var field extractedFieldResponse
	err = h.DB.QueryRow(ctx, `
SELECT
id, pa_request_id, document_id, field_key, field_value,
confidence, evidence_text, verification_status,
reviewed_by, reviewed_at, created_at
FROM extracted_fields
WHERE id = $1 AND pa_request_id = $2
`, fieldID, requestID).Scan(
		&field.ID,
		&field.PARequestID,
		&field.DocumentID,
		&field.FieldKey,
		&field.FieldValue,
		&field.Confidence,
		&field.EvidenceText,
		&field.VerificationStatus,
		&field.ReviewedBy,
		&field.ReviewedAt,
		&field.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Field diperbarui, tetapi gagal mengambil hasilnya."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Extracted field berhasil diperbarui.",
		"data":    field,
	})
}

func (h *ExtractedFieldHandler) canAccessRequest(
	ctx context.Context,
	c *gin.Context,
	requestID int64,
) (bool, error) {
	var createdBy int64
	err := h.DB.QueryRow(ctx, `
SELECT created_by
FROM pa_requests
WHERE id = $1 AND deleted_at IS NULL
`, requestID).Scan(&createdBy)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	role := c.GetString("role")
	return role == "admin" ||
		role == "reviewer" ||
		(role == "staff" && c.GetInt64("user_id") == createdBy), nil
}
