package handlers

import (
	"net/http"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecklistHandler struct {
	pool *pgxpool.Pool
}

func NewChecklistHandler(pool *pgxpool.Pool) *ChecklistHandler {
	return &ChecklistHandler{pool: pool}
}

func (h *ChecklistHandler) Evaluate(c *gin.Context) {
	if c.GetString("role") != "reviewer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Evaluasi checklist dilakukan oleh Reviewer."})
		return
	}
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	if !canAccessChecklistRequest(c, h.pool, requestID) {
		return
	}
	var status string
	if err := h.pool.QueryRow(c.Request.Context(), `SELECT status FROM pa_requests WHERE id=$1 AND deleted_at IS NULL`, requestID).Scan(&status); err != nil || status != "in_review" {
		c.JSON(http.StatusConflict, gin.H{"error": "Checklist hanya dapat dievaluasi saat permohonan berstatus in_review."})
		return
	}

	result, err := services.EvaluateChecklist(
		c.Request.Context(),
		h.pool,
		requestID,
		c.GetInt64("user_id"),
	)
	if err != nil {
		switch err.Error() {
		case "PA request not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "PA request has no checklist version assigned",
			"checklist version has no required field rules":
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to evaluate checklist",
			})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ChecklistHandler) Results(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	if !canAccessChecklistRequest(c, h.pool, requestID) {
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT
			cr.id,
			cr.checklist_rule_id,
			r.rule_key,
			r.description,
			r.required_field_key,
			cr.status,
			cr.reason,
			cr.evidence,
			cr.updated_at
		FROM checklist_results cr
		JOIN checklist_rules r ON r.id = cr.checklist_rule_id
		JOIN pa_requests pr ON pr.id = cr.pa_request_id
		WHERE cr.pa_request_id = $1
		  AND pr.deleted_at IS NULL
		ORDER BY r.id
	`, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve checklist results",
		})
		return
	}
	defer rows.Close()

	type resultItem struct {
		ID          int64     `json:"id"`
		RuleID      int64     `json:"rule_id"`
		RuleKey     string    `json:"rule_key"`
		Description string    `json:"description"`
		FieldKey    *string   `json:"field_key"`
		Status      string    `json:"status"`
		Reason      string    `json:"reason"`
		Evidence    *string   `json:"evidence"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	results := make([]resultItem, 0)

	for rows.Next() {
		var item resultItem
		if err := rows.Scan(
			&item.ID,
			&item.RuleID,
			&item.RuleKey,
			&item.Description,
			&item.FieldKey,
			&item.Status,
			&item.Reason,
			&item.Evidence,
			&item.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read checklist result",
			})
			return
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read checklist results",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pa_request_id": requestID,
		"total":         len(results),
		"results":       results,
	})
}

func canAccessChecklistRequest(
	c *gin.Context,
	pool *pgxpool.Pool,
	requestID int64,
) bool {
	userValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return false
	}

	roleValue, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User role unavailable"})
		return false
	}

	role, ok := roleValue.(string)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user role"})
		return false
	}

	var userID int64
	if role == "staff" {
		userID, ok = userValue.(int64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identity"})
			return false
		}
	}

	var existsRequest bool
	var err error

	if role == "admin" || role == "reviewer" {
		err = pool.QueryRow(c.Request.Context(), `
			SELECT EXISTS (
				SELECT 1 FROM pa_requests
				WHERE id = $1 AND deleted_at IS NULL
			)
		`, requestID).Scan(&existsRequest)
	} else if role == "staff" {
		err = pool.QueryRow(c.Request.Context(), `
			SELECT EXISTS (
				SELECT 1 FROM pa_requests
				WHERE id = $1
				  AND created_by = $2
				  AND deleted_at IS NULL
			)
		`, requestID, userID).Scan(&existsRequest)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return false
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify PA request",
		})
		return false
	}
	if !existsRequest {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request not found"})
		return false
	}

	return true
}
