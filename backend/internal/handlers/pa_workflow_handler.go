package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PAWorkflowHandler struct{ DB *pgxpool.Pool }

type paTransitionInput struct {
	Action            string `json:"action" binding:"required"`
	Note              string `json:"note"`
	PayerReference    string `json:"payer_reference"`
	AppealReference   string `json:"appeal_reference"`
	SubmissionChannel string `json:"submission_channel"`
	PayerDecision     string `json:"payer_decision"`
	PayerDecisionNote string `json:"payer_decision_note"`
}

type requestWorkflowState struct {
	ID                   int64
	CreatedBy            int64
	RequestNumber        string
	PayerName            string
	ServiceType          string
	Status               string
	PayerDecision        *string
	AppealDecision       *string
	ChecklistVersion     *int64
	OrderingProviderNPI  string
	ServicingProviderNPI string
}

func NewPAWorkflowHandler(db *pgxpool.Pool) *PAWorkflowHandler {
	return &PAWorkflowHandler{DB: db}
}

func isNumericNPI(value string) bool {
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func findActiveChecklistVersion(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, payerName, serviceType string) (*int64, error) {
	var id int64
	err := queryer.QueryRow(ctx, `
		SELECT id FROM checklist_versions
		WHERE is_active=TRUE
		  AND lower(btrim(payer_name))=lower(btrim($1))
		  AND lower(btrim(service_type))=lower(btrim($2))
		ORDER BY version_number DESC, effective_date DESC NULLS LAST, id DESC
		LIMIT 1
	`, payerName, serviceType).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (h *PAWorkflowHandler) Transition(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	var input paTransitionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aksi alur tidak valid."})
		return
	}
	input.Action = strings.TrimSpace(input.Action)
	input.Note = strings.TrimSpace(input.Note)
	input.PayerReference = strings.TrimSpace(input.PayerReference)
	input.AppealReference = strings.TrimSpace(input.AppealReference)
	input.SubmissionChannel = strings.TrimSpace(input.SubmissionChannel)
	if input.SubmissionChannel == "" {
		input.SubmissionChannel = "unknown"
	}
	input.PayerDecision = strings.TrimSpace(input.PayerDecision)
	input.PayerDecisionNote = strings.TrimSpace(input.PayerDecisionNote)
	role, actorID := c.GetString("role"), c.GetInt64("user_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		log.Printf("begin PA request %d transition failed: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Perubahan status gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var request requestWorkflowState
	err = tx.QueryRow(ctx, `
		SELECT id, created_by, request_number, payer_name, service_type, status, payer_decision, appeal_decision, checklist_version_id,
		       COALESCE(ordering_provider_npi, provider_npi, ''), COALESCE(servicing_provider_npi, '')
		FROM pa_requests WHERE id=$1 AND deleted_at IS NULL FOR UPDATE
	`, requestID).Scan(&request.ID, &request.CreatedBy, &request.RequestNumber, &request.PayerName, &request.ServiceType, &request.Status, &request.PayerDecision, &request.AppealDecision, &request.ChecklistVersion, &request.OrderingProviderNPI, &request.ServicingProviderNPI)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("load PA request %d for transition failed: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status permohonan gagal dimuat."})
		return
	}
	if role == "staff" && actorID != request.CreatedBy {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan tidak ditemukan."})
		return
	}

	fromStatus, toStatus := request.Status, ""
	switch input.Action {
	case "submit_for_review":
		if role != "staff" || (fromStatus != "draft" && fromStatus != "needs_information" && fromStatus != "payer_needs_information") {
			c.JSON(http.StatusConflict, gin.H{"error": "Hanya Staff pemilik yang dapat mengirim draft atau permohonan yang dikembalikan."})
			return
		}
		if request.ChecklistVersion != nil {
			var matches bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM checklist_versions WHERE id=$1 AND lower(btrim(payer_name))=lower(btrim($2)) AND lower(btrim(service_type))=lower(btrim($3)))`, *request.ChecklistVersion, request.PayerName, request.ServiceType).Scan(&matches); err != nil {
				log.Printf("check PA request %d checklist match for transition failed: %v", requestID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Kecocokan checklist gagal diperiksa."})
				return
			}
			if !matches {
				request.ChecklistVersion = nil
			}
		}
		if request.ChecklistVersion == nil {
			request.ChecklistVersion, err = findActiveChecklistVersion(ctx, tx, request.PayerName, request.ServiceType)
			if err != nil {
				log.Printf("find active checklist for PA request %d transition failed: %v", requestID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist aktif gagal dicari."})
				return
			}
			if request.ChecklistVersion == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Checklist aktif untuk payer dan layanan ini belum tersedia. Admin perlu membuat dan mengaktifkannya dari Kelola checklist."})
				return
			}
			if _, err = tx.Exec(ctx, `UPDATE pa_requests SET checklist_version_id=$2 WHERE id=$1`, request.ID, *request.ChecklistVersion); err != nil {
				log.Printf("attach checklist to PA request %d during transition failed: %v", requestID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist gagal dipasangkan ke permohonan."})
				return
			}
		}
		var requiredRules, completedDocuments, totalDocuments int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM checklist_rules WHERE checklist_version_id=$1 AND is_required=TRUE AND required_field_key IS NOT NULL`, *request.ChecklistVersion).Scan(&requiredRules); err != nil {
			log.Printf("count required checklist rules for PA request %d transition failed: %v", requestID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Aturan checklist gagal diperiksa."})
			return
		}
		if requiredRules == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Checklist terpasang tetapi belum memiliki aturan field wajib. Admin perlu memperbaiki checklist."})
			return
		}
		var orderingRequired, servicingRequired bool
		if err = tx.QueryRow(ctx, `
			SELECT
				COALESCE(bool_or(lower(replace(required_field_key, '-', '_')) IN ('provider_npi', 'ordering_provider_npi')), FALSE),
				COALESCE(bool_or(lower(replace(required_field_key, '-', '_')) = 'servicing_provider_npi'), FALSE)
			FROM checklist_rules
			WHERE checklist_version_id=$1 AND is_required=TRUE AND required_field_key IS NOT NULL
		`, *request.ChecklistVersion).Scan(&orderingRequired, &servicingRequired); err != nil {
			log.Printf("check provider-role requirements for PA request %d failed: %v", requestID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Persyaratan provider pada checklist gagal diperiksa."})
			return
		}
		missingProviderRoles := make([]string, 0, 2)
		if orderingRequired && (len(request.OrderingProviderNPI) != 10 || !isNumericNPI(request.OrderingProviderNPI)) {
			missingProviderRoles = append(missingProviderRoles, "Ordering provider NPI")
		}
		if servicingRequired && (len(request.ServicingProviderNPI) != 10 || !isNumericNPI(request.ServicingProviderNPI)) {
			missingProviderRoles = append(missingProviderRoles, "Servicing provider NPI")
		}
		if len(missingProviderRoles) > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Lengkapi NPI 10 digit untuk peran yang diwajibkan checklist: " + strings.Join(missingProviderRoles, ", ") + "."})
			return
		}
		if err = tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE parsing_status='completed') FROM documents WHERE pa_request_id=$1`, request.ID).Scan(&totalDocuments, &completedDocuments); err != nil {
			log.Printf("count documents for PA request %d transition failed: %v", requestID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Status dokumen gagal diperiksa."})
			return
		}
		if totalDocuments == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Unggah dan parsing minimal satu dokumen sebelum mengirim permohonan untuk review."})
			return
		}
		if completedDocuments != totalDocuments {
			c.JSON(http.StatusConflict, gin.H{"error": "Semua dokumen harus berhasil diparsing sebelum permohonan dikirim untuk review."})
			return
		}
		toStatus = "ready_for_review"
	case "start_review":
		if role != "reviewer" || fromStatus != "ready_for_review" {
			c.JSON(http.StatusConflict, gin.H{"error": "Reviewer hanya dapat memulai permohonan yang sudah masuk antrian review."})
			return
		}
		toStatus = "in_review"
	case "request_information":
		if role != "reviewer" || fromStatus != "in_review" {
			c.JSON(http.StatusConflict, gin.H{"error": "Hanya Reviewer yang sedang memeriksa dapat meminta informasi tambahan."})
			return
		}
		if input.Note == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tuliskan informasi atau dokumen yang perlu dilengkapi Staff."})
			return
		}
		toStatus = "needs_information"
	case "complete_review":
		if role != "reviewer" || fromStatus != "in_review" {
			c.JSON(http.StatusConflict, gin.H{"error": "Hanya Reviewer yang sedang memeriksa dapat menyelesaikan review."})
			return
		}
		evaluation, evalErr := services.EvaluateChecklistInTx(ctx, tx, request.ID, actorID)
		if evalErr != nil {
			log.Printf("evaluate checklist for PA request %d during transition failed: %v", requestID, evalErr)
			c.JSON(http.StatusConflict, gin.H{"error": "Checklist belum dapat dievaluasi: " + evalErr.Error()})
			return
		}
		if evaluation.Summary.Total == 0 || evaluation.Summary.Missing > 0 || evaluation.Summary.NeedsReview > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Review belum bisa diselesaikan. Pastikan semua field wajib sudah diverifikasi dan checklist berstatus fulfilled.", "checklist_summary": evaluation.Summary})
			return
		}
		toStatus = "ready_for_submission"
	case "record_submission":
		if role != "staff" || actorID != request.CreatedBy || fromStatus != "ready_for_submission" {
			c.JSON(http.StatusConflict, gin.H{"error": "Staff pemilik hanya dapat mencatat pengajuan ke payer setelah review selesai."})
			return
		}
		switch input.SubmissionChannel {
		case "payer_portal", "clearinghouse", "phone", "fax", "mail", "api_manual", "other", "unknown":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih kanal pengajuan yang digunakan."})
			return
		}
		toStatus = "submitted"
	case "record_decision":
		if role != "staff" || actorID != request.CreatedBy || fromStatus != "submitted" {
			c.JSON(http.StatusConflict, gin.H{"error": "Staff pemilik hanya dapat mencatat keputusan untuk permohonan yang sudah diajukan."})
			return
		}
		switch input.PayerDecision {
		case "approved", "denied", "partially_approved", "withdrawn":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih outcome payer: approved, denied, partially_approved, atau withdrawn."})
			return
		}
		if input.PayerDecisionNote == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Catatan keputusan payer wajib diisi."})
			return
		}
		toStatus = "completed"
	case "request_payer_information":
		if role != "staff" || actorID != request.CreatedBy || fromStatus != "submitted" {
			c.JSON(http.StatusConflict, gin.H{"error": "Staff pemilik hanya dapat mencatat permintaan informasi payer untuk permohonan yang sudah diajukan."})
			return
		}
		if input.Note == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Catat informasi atau dokumen tambahan yang diminta payer."})
			return
		}
		toStatus = "payer_needs_information"
	case "record_appeal_submission":
		if role != "staff" || actorID != request.CreatedBy || fromStatus != "completed" || request.PayerDecision == nil || (*request.PayerDecision != "denied" && *request.PayerDecision != "partially_approved") {
			c.JSON(http.StatusConflict, gin.H{"error": "Banding hanya dapat dicatat Staff pemilik untuk keputusan payer yang ditolak atau disetujui sebagian."})
			return
		}
		if request.AppealDecision != nil && *request.AppealDecision != "denied" && *request.AppealDecision != "partially_approved" {
			c.JSON(http.StatusConflict, gin.H{"error": "Banding lanjutan hanya dapat dicatat setelah banding sebelumnya ditolak atau disetujui sebagian."})
			return
		}
		if input.Note == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Catat alasan dan ringkasan banding."})
			return
		}
		switch input.SubmissionChannel {
		case "payer_portal", "clearinghouse", "phone", "fax", "mail", "api_manual", "other", "unknown":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih kanal pengajuan banding yang digunakan."})
			return
		}
		toStatus = "appeal_submitted"
	case "record_appeal_decision":
		if role != "staff" || actorID != request.CreatedBy || fromStatus != "appeal_submitted" {
			c.JSON(http.StatusConflict, gin.H{"error": "Staff pemilik hanya dapat mencatat respons atas banding yang sudah diajukan."})
			return
		}
		switch input.PayerDecision {
		case "approved", "denied", "partially_approved", "withdrawn":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih outcome banding: approved, denied, partially_approved, atau withdrawn."})
			return
		}
		if input.PayerDecisionNote == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Catatan respons atas banding wajib diisi."})
			return
		}
		toStatus = "completed"
	case "cancel":
		if role != "staff" || actorID != request.CreatedBy || (fromStatus != "draft" && fromStatus != "needs_information") {
			c.JSON(http.StatusConflict, gin.H{"error": "Hanya Staff pemilik yang dapat membatalkan draft atau permohonan yang dikembalikan."})
			return
		}
		if input.Note == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Alasan pembatalan wajib diisi."})
			return
		}
		toStatus = "cancelled"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aksi alur tidak dikenal."})
		return
	}

	var updatedStatus string
	if input.Action == "record_submission" {
		err = tx.QueryRow(ctx, `UPDATE pa_requests SET status=$2,payer_reference=COALESCE(NULLIF($3,''),payer_reference),submitted_at=NOW(),updated_at=NOW() WHERE id=$1 RETURNING status`, request.ID, toStatus, input.PayerReference).Scan(&updatedStatus)
	} else if input.Action == "record_decision" {
		err = tx.QueryRow(ctx, `UPDATE pa_requests SET status=$2,payer_decision=$3,payer_decision_note=$4,payer_decision_at=NOW(),updated_at=NOW() WHERE id=$1 RETURNING status`, request.ID, toStatus, input.PayerDecision, input.PayerDecisionNote).Scan(&updatedStatus)
	} else if input.Action == "record_appeal_submission" {
		err = tx.QueryRow(ctx, `UPDATE pa_requests SET status=$2,appeal_reference=NULLIF($3,''),appeal_submitted_at=NOW(),appeal_decision=NULL,appeal_decision_at=NULL,appeal_decision_note=NULL,updated_at=NOW() WHERE id=$1 RETURNING status`, request.ID, toStatus, input.AppealReference).Scan(&updatedStatus)
	} else if input.Action == "record_appeal_decision" {
		err = tx.QueryRow(ctx, `UPDATE pa_requests SET status=$2,appeal_decision=$3,appeal_decision_at=NOW(),appeal_decision_note=$4,updated_at=NOW() WHERE id=$1 RETURNING status`, request.ID, toStatus, input.PayerDecision, input.PayerDecisionNote).Scan(&updatedStatus)
	} else {
		err = tx.QueryRow(ctx, `UPDATE pa_requests SET status=$2,updated_at=NOW() WHERE id=$1 RETURNING status`, request.ID, toStatus).Scan(&updatedStatus)
	}
	if err != nil {
		log.Printf("update PA request %d status (%s -> %s) failed: %v", request.ID, fromStatus, toStatus, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status permohonan gagal diperbarui."})
		return
	}
	historyNote := firstNonEmpty(input.Note, input.PayerDecisionNote)
	if input.Action == "record_submission" {
		if input.SubmissionChannel == "unknown" {
			historyNote = "Pengajuan dicatat; kanal tidak dicatat."
		} else {
			historyNote = "Pengajuan dicatat melalui " + input.SubmissionChannel
		}
		if input.PayerReference != "" {
			historyNote += "; referensi payer: " + input.PayerReference
		}
	} else if input.Action == "record_appeal_submission" {
		if input.SubmissionChannel == "unknown" {
			historyNote = "Banding dicatat; kanal tidak dicatat. " + input.Note
		} else {
			historyNote = "Banding dicatat melalui " + input.SubmissionChannel + ". " + input.Note
		}
		if input.AppealReference != "" {
			historyNote += "; referensi banding: " + input.AppealReference
		}
	} else if input.Action == "record_decision" || input.Action == "record_appeal_decision" {
		historyNote = input.PayerDecision + ": " + input.PayerDecisionNote
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO pa_request_status_history
			(pa_request_id, actor_id, from_status, to_status, action, note, details)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), jsonb_build_object('payer_reference', NULLIF($7, ''), 'appeal_reference', NULLIF($8, ''), 'submission_channel', NULLIF($9, ''), 'decision', NULLIF($10, '')))
	`, request.ID, actorID, fromStatus, toStatus, input.Action, historyNote, input.PayerReference, input.AppealReference, input.SubmissionChannel, input.PayerDecision)
	if err != nil {
		log.Printf("save PA request %d status history (%s -> %s) failed: %v", request.ID, fromStatus, toStatus, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat status gagal disimpan."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		log.Printf("commit PA request %d transition (%s -> %s) failed: %v", request.ID, fromStatus, toStatus, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status permohonan gagal disimpan."})
		return
	}
	data := gin.H{"id": request.ID, "status": updatedStatus}
	if input.Action == "record_appeal_decision" {
		data["appeal_decision"] = nullableString(input.PayerDecision)
	} else {
		data["payer_decision"] = nullableString(input.PayerDecision)
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "message": "Tahap permohonan diperbarui."})
}

func (h *PAWorkflowHandler) History(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var createdBy int64
	if err := h.DB.QueryRow(ctx, `SELECT created_by FROM pa_requests WHERE id=$1 AND deleted_at IS NULL`, requestID).Scan(&createdBy); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan tidak ditemukan."})
		return
	}
	role := c.GetString("role")
	if role == "staff" && c.GetInt64("user_id") != createdBy {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan tidak ditemukan."})
		return
	}
	if role != "staff" && role != "reviewer" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak memiliki akses."})
		return
	}
	rows, err := h.DB.Query(ctx, `
		SELECT h.id,h.from_status,h.to_status,h.action,h.note,h.created_at,u.full_name
		FROM pa_request_status_history h
		LEFT JOIN users u ON u.id=h.actor_id
		WHERE h.pa_request_id=$1 ORDER BY h.created_at,h.id
	`, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat permohonan gagal dimuat."})
		return
	}
	defer rows.Close()
	type historyRecord struct {
		ID         int64     `json:"id"`
		FromStatus *string   `json:"from_status"`
		ToStatus   string    `json:"to_status"`
		Action     string    `json:"action"`
		Note       *string   `json:"note"`
		CreatedAt  time.Time `json:"created_at"`
		ActorName  *string   `json:"actor_name"`
	}
	items := make([]historyRecord, 0)
	for rows.Next() {
		var item historyRecord
		if err := rows.Scan(&item.ID, &item.FromStatus, &item.ToStatus, &item.Action, &item.Note, &item.CreatedAt, &item.ActorName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat permohonan gagal dibaca."})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat permohonan gagal dibaca."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
