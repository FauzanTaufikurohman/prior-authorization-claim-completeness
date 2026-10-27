package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecklistVersionHandler struct{ DB *pgxpool.Pool }

type checklistVersionInput struct {
	Name          string               `json:"name"`
	PayerName     string               `json:"payer_name"`
	ServiceType   string               `json:"service_type"`
	VersionNumber int                  `json:"version_number"`
	SourceURL     *string              `json:"source_url"`
	EffectiveDate *string              `json:"effective_date"`
	Rules         []checklistRuleInput `json:"rules"`
}

type checklistRuleInput struct {
	RuleKey          string  `json:"rule_key"`
	Description      string  `json:"description"`
	RequiredFieldKey *string `json:"required_field_key"`
	IsRequired       bool    `json:"is_required"`
}

type checklistVersionRecord struct {
	ID            int64                 `json:"id"`
	Name          string                `json:"name"`
	PayerName     string                `json:"payer_name"`
	ServiceType   string                `json:"service_type"`
	VersionNumber int                   `json:"version_number"`
	SourceURL     *string               `json:"source_url"`
	EffectiveDate *string               `json:"effective_date"`
	ReviewedBy    *int64                `json:"reviewed_by"`
	IsActive      bool                  `json:"is_active"`
	CreatedAt     time.Time             `json:"created_at"`
	Rules         []checklistRuleRecord `json:"rules"`
}

type checklistRuleRecord struct {
	ID               int64   `json:"id"`
	RuleKey          string  `json:"rule_key"`
	Description      string  `json:"description"`
	RequiredFieldKey *string `json:"required_field_key"`
	IsRequired       bool    `json:"is_required"`
}

type activeChecklistOption struct {
	ID            int64                 `json:"id"`
	Name          string                `json:"name"`
	PayerName     string                `json:"payer_name"`
	ServiceType   string                `json:"service_type"`
	VersionNumber int                   `json:"version_number"`
	Rules         []checklistRuleRecord `json:"rules"`
}

func NewChecklistVersionHandler(db *pgxpool.Pool) *ChecklistVersionHandler {
	return &ChecklistVersionHandler{DB: db}
}

func (h *ChecklistVersionHandler) Options(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	rows, err := h.DB.Query(ctx, `
		SELECT v.id, v.name, v.payer_name, v.service_type, v.version_number,
		       r.id, r.rule_key, r.description, r.required_field_key, r.is_required
		FROM checklist_versions v
		JOIN checklist_rules r ON r.checklist_version_id=v.id
		WHERE v.is_active=TRUE
		ORDER BY lower(v.payer_name), lower(v.service_type), r.id
	`)
	if err != nil {
		log.Printf("list active checklist options failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pilihan payer dan layanan gagal dimuat."})
		return
	}
	defer rows.Close()
	options := make([]activeChecklistOption, 0)
	indexes := make(map[int64]int)
	for rows.Next() {
		var option activeChecklistOption
		var ruleID int64
		var rule checklistRuleRecord
		if err := rows.Scan(&option.ID, &option.Name, &option.PayerName, &option.ServiceType, &option.VersionNumber,
			&ruleID, &rule.RuleKey, &rule.Description, &rule.RequiredFieldKey, &rule.IsRequired); err != nil {
			log.Printf("scan active checklist option failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Pilihan payer dan layanan gagal dibaca."})
			return
		}
		index, exists := indexes[option.ID]
		if !exists {
			option.Rules = make([]checklistRuleRecord, 0)
			options = append(options, option)
			index = len(options) - 1
			indexes[option.ID] = index
		}
		if ruleID != 0 {
			rule.ID = ruleID
			options[index].Rules = append(options[index].Rules, rule)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("iterate active checklist options failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pilihan payer dan layanan gagal dibaca."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": options})
}

func (h *ChecklistVersionHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	rows, err := h.DB.Query(ctx, `
		SELECT id, name, payer_name, service_type, version_number,
		       source_url, effective_date::text, reviewed_by, is_active, created_at
		FROM checklist_versions
		ORDER BY lower(payer_name), lower(service_type), version_number DESC
	`)
	if err != nil {
		log.Printf("list checklist versions query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar checklist gagal dimuat."})
		return
	}
	versions := make([]checklistVersionRecord, 0)
	for rows.Next() {
		var version checklistVersionRecord
		if err := rows.Scan(&version.ID, &version.Name, &version.PayerName, &version.ServiceType,
			&version.VersionNumber, &version.SourceURL, &version.EffectiveDate, &version.ReviewedBy,
			&version.IsActive, &version.CreatedAt); err != nil {
			rows.Close()
			log.Printf("scan checklist version failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar checklist gagal dibaca."})
			return
		}
		version.Rules = make([]checklistRuleRecord, 0)
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("iterate checklist versions failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Daftar checklist gagal dibaca."})
		return
	}
	rows.Close()
	for i := range versions {
		ruleRows, err := h.DB.Query(ctx, `
			SELECT id, rule_key, description, required_field_key, is_required
			FROM checklist_rules WHERE checklist_version_id=$1 ORDER BY id
		`, versions[i].ID)
		if err != nil {
			log.Printf("list rules for checklist version %d failed: %v", versions[i].ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Aturan checklist gagal dimuat."})
			return
		}
		for ruleRows.Next() {
			var rule checklistRuleRecord
			if err := ruleRows.Scan(&rule.ID, &rule.RuleKey, &rule.Description, &rule.RequiredFieldKey, &rule.IsRequired); err != nil {
				ruleRows.Close()
				log.Printf("scan rule for checklist version %d failed: %v", versions[i].ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Aturan checklist gagal dibaca."})
				return
			}
			versions[i].Rules = append(versions[i].Rules, rule)
		}
		if err := ruleRows.Err(); err != nil {
			ruleRows.Close()
			log.Printf("iterate rules for checklist version %d failed: %v", versions[i].ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Aturan checklist gagal dibaca."})
			return
		}
		ruleRows.Close()
	}
	c.JSON(http.StatusOK, gin.H{"data": versions})
}

func (h *ChecklistVersionHandler) Create(c *gin.Context) {
	var input checklistVersionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format checklist tidak valid."})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.PayerName = strings.TrimSpace(input.PayerName)
	input.ServiceType = strings.TrimSpace(input.ServiceType)
	if input.Name == "" || input.PayerName == "" || input.ServiceType == "" || input.VersionNumber < 1 || len(input.Rules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, payer, layanan, nomor versi positif, dan minimal satu aturan wajib diisi."})
		return
	}
	seen := make(map[string]struct{}, len(input.Rules))
	for i := range input.Rules {
		rule := &input.Rules[i]
		rule.RuleKey = strings.TrimSpace(rule.RuleKey)
		rule.Description = strings.TrimSpace(rule.Description)
		if rule.RequiredFieldKey != nil {
			value := strings.TrimSpace(*rule.RequiredFieldKey)
			rule.RequiredFieldKey = &value
		}
		if rule.RuleKey == "" || rule.Description == "" || (rule.IsRequired && (rule.RequiredFieldKey == nil || *rule.RequiredFieldKey == "")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Setiap aturan harus memiliki key dan deskripsi; aturan wajib harus terhubung ke field."})
			return
		}
		if _, exists := seen[rule.RuleKey]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Key aturan tidak boleh duplikat dalam satu versi."})
			return
		}
		seen[rule.RuleKey] = struct{}{}
	}
	if input.EffectiveDate != nil {
		date := strings.TrimSpace(*input.EffectiveDate)
		if date == "" {
			input.EffectiveDate = nil
		} else if _, err := time.Parse("2006-01-02", date); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tanggal efektif harus menggunakan format YYYY-MM-DD."})
			return
		} else {
			input.EffectiveDate = &date
		}
	}
	if input.SourceURL != nil {
		value := strings.TrimSpace(*input.SourceURL)
		if value == "" {
			input.SourceURL = nil
		} else {
			input.SourceURL = &value
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pembuatan checklist gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var versionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO checklist_versions (name,payer_name,service_type,version_number,source_url,effective_date,is_active)
		VALUES ($1,$2,$3,$4,$5,$6,FALSE)
		RETURNING id
	`, input.Name, input.PayerName, input.ServiceType, input.VersionNumber, input.SourceURL, input.EffectiveDate).Scan(&versionID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Versi checklist gagal dibuat. Periksa apakah nomor versi untuk payer dan layanan ini sudah digunakan."})
		return
	}
	for _, rule := range input.Rules {
		_, err = tx.Exec(ctx, `
			INSERT INTO checklist_rules (checklist_version_id,rule_key,description,required_field_key,is_required)
			VALUES ($1,$2,$3,$4,$5)
		`, versionID, rule.RuleKey, rule.Description, rule.RequiredFieldKey, rule.IsRequired)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Aturan checklist gagal disimpan. Gunakan key field yang unik."})
			return
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO checklist_version_audit_logs (actor_id,checklist_version_id,action,details)
		VALUES ($1,$2,'created',jsonb_build_object('payer_name',$3::text,'service_type',$4::text,'version_number',$5::integer))
	`, c.GetInt64("user_id"), versionID, input.PayerName, input.ServiceType, input.VersionNumber)
	if err != nil {
		log.Printf("record checklist version %d creation audit failed: %v", versionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat pembuatan checklist gagal disimpan."})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist gagal disimpan."})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": versionID, "name": input.Name, "payer_name": input.PayerName, "service_type": input.ServiceType, "version_number": input.VersionNumber, "is_active": false}})
}

func (h *ChecklistVersionHandler) Activate(c *gin.Context) {
	versionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || versionID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID versi checklist tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivasi checklist gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(9137284)`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivasi checklist gagal diproses."})
		return
	}
	var payer, service string
	err = tx.QueryRow(ctx, `SELECT payer_name, service_type FROM checklist_versions WHERE id=$1 FOR UPDATE`, versionID).Scan(&payer, &service)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versi checklist tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Versi checklist gagal dimuat."})
		return
	}
	var ruleCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM checklist_rules WHERE checklist_version_id=$1 AND is_required=TRUE AND required_field_key IS NOT NULL`, versionID).Scan(&ruleCount); err != nil || ruleCount == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Checklist harus memiliki minimal satu aturan wajib yang terhubung ke field sebelum diaktifkan."})
		return
	}
	_, err = tx.Exec(ctx, `UPDATE checklist_versions SET is_active=FALSE WHERE lower(btrim(payer_name))=lower(btrim($1)) AND lower(btrim(service_type))=lower(btrim($2)) AND id<>$3`, payer, service, versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Versi checklist lama gagal dinonaktifkan."})
		return
	}
	_, err = tx.Exec(ctx, `UPDATE checklist_versions SET is_active=TRUE, reviewed_by=$2 WHERE id=$1`, versionID, c.GetInt64("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist gagal diaktifkan."})
		return
	}
	_, err = tx.Exec(ctx, `INSERT INTO checklist_version_audit_logs (actor_id,checklist_version_id,action) VALUES ($1,$2,'activated')`, c.GetInt64("user_id"), versionID)
	if err != nil {
		log.Printf("record checklist version %d activation audit failed: %v", versionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat aktivasi checklist gagal disimpan."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivasi checklist gagal disimpan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Checklist aktif untuk payer dan layanan ini.", "id": versionID})
}

func (h *ChecklistVersionHandler) Deactivate(c *gin.Context) {
	versionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || versionID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID versi checklist tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penonaktifan checklist gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(9137284)`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penonaktifan checklist gagal diproses."})
		return
	}
	var active bool
	err = tx.QueryRow(ctx, `SELECT is_active FROM checklist_versions WHERE id=$1 FOR UPDATE`, versionID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versi checklist tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Versi checklist gagal dimuat."})
		return
	}
	if !active {
		c.JSON(http.StatusOK, gin.H{"message": "Checklist sudah nonaktif.", "id": versionID})
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE checklist_versions SET is_active=FALSE WHERE id=$1`, versionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist gagal dinonaktifkan."})
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO checklist_version_audit_logs (actor_id,checklist_version_id,action) VALUES ($1,$2,'deactivated')`, c.GetInt64("user_id"), versionID); err != nil {
		log.Printf("record checklist version %d deactivation audit failed: %v", versionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Riwayat penonaktifan checklist gagal disimpan."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penonaktifan checklist gagal disimpan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Checklist dinonaktifkan. Checklist ini tidak akan muncul untuk permohonan baru.", "id": versionID})
}

func (h *ChecklistVersionHandler) Delete(c *gin.Context) {
	versionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || versionID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID versi checklist tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penghapusan checklist gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	err = tx.QueryRow(ctx, `SELECT is_active FROM checklist_versions WHERE id=$1 FOR UPDATE`, versionID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Versi checklist tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Versi checklist gagal dimuat."})
		return
	}
	if active {
		c.JSON(http.StatusConflict, gin.H{"error": "Nonaktifkan checklist sebelum menghapusnya."})
		return
	}
	var inUse bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pa_requests WHERE checklist_version_id=$1)`, versionID).Scan(&inUse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pemakaian checklist gagal diperiksa."})
		return
	}
	if inUse {
		c.JSON(http.StatusConflict, gin.H{"error": "Checklist sudah terhubung ke permohonan. Nonaktifkan saja agar riwayat permohonan tetap utuh."})
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM checklist_versions WHERE id=$1 AND is_active=FALSE`, versionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Checklist gagal dihapus."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penghapusan checklist gagal disimpan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Checklist berhasil dihapus.", "id": versionID})
}
