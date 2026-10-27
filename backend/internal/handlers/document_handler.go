package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxMultipartOverhead int64 = 1 * 1024 * 1024

type DocumentHandler struct {
	DB          *pgxpool.Pool
	StorageRoot string
}

func NewDocumentHandler(db *pgxpool.Pool) *DocumentHandler {
	root := strings.TrimSpace(os.Getenv("DOCUMENT_STORAGE_DIR"))
	if root == "" {
		root = filepath.Join("var", "private-documents")
	}

	return &DocumentHandler{
		DB:          db,
		StorageRoot: root,
	}
}

type documentMetadata struct {
	ID            int64     `json:"id"`
	PARequestID   int64     `json:"pa_request_id"`
	OriginalName  string    `json:"original_filename"`
	MIMEType      string    `json:"mime_type"`
	SizeBytes     int64     `json:"size_bytes"`
	ParsingStatus string    `json:"parsing_status"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *DocumentHandler) List(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database service unavailable.",
		})
		return
	}

	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var createdBy int64
	err := h.DB.QueryRow(ctx, `
		SELECT created_by
		FROM pa_requests
		WHERE id = $1
		  AND deleted_at IS NULL
	`, requestID).Scan(&createdBy)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memeriksa PA request.",
		})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt64("user_id")
	if role != "admin" &&
		role != "reviewer" &&
		!(role == "staff" && userID == createdBy) {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}

	rows, err := h.DB.Query(ctx, `
		SELECT id, pa_request_id, original_filename, mime_type,
		       size_bytes, parsing_status, created_at
		FROM documents
		WHERE pa_request_id = $1
		ORDER BY created_at DESC, id DESC
	`, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil metadata dokumen.",
		})
		return
	}
	defer rows.Close()

	documents := make([]documentMetadata, 0)
	for rows.Next() {
		var document documentMetadata
		if err := rows.Scan(
			&document.ID,
			&document.PARequestID,
			&document.OriginalName,
			&document.MIMEType,
			&document.SizeBytes,
			&document.ParsingStatus,
			&document.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal membaca metadata dokumen.",
			})
			return
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyelesaikan pembacaan metadata dokumen.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": documents})
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	documentID, err := strconv.ParseInt(c.Param("documentId"), 10, 64)
	if err != nil || documentID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penghapusan dokumen gagal dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var createdBy int64
	var requestStatus string
	err = tx.QueryRow(ctx, `SELECT created_by, status FROM pa_requests WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, requestID).Scan(&createdBy, &requestStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PA request gagal diperiksa."})
		return
	}
	if c.GetString("role") != "staff" || c.GetInt64("user_id") != createdBy {
		c.JSON(http.StatusNotFound, gin.H{"error": "PA request tidak ditemukan."})
		return
	}
	if requestStatus != "draft" && requestStatus != "needs_information" && requestStatus != "payer_needs_information" {
		c.JSON(http.StatusConflict, gin.H{"error": "Dokumen hanya dapat dihapus saat draft atau ketika Staff sedang melengkapi permintaan informasi."})
		return
	}
	var storageKey, parsingStatus string
	err = tx.QueryRow(ctx, `SELECT storage_key, parsing_status FROM documents WHERE id=$1 AND pa_request_id=$2 FOR UPDATE`, documentID, requestID).Scan(&storageKey, &parsingStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Metadata dokumen gagal dimuat."})
		return
	}
	if parsingStatus == "processing" {
		c.JSON(http.StatusConflict, gin.H{"error": "Dokumen sedang diproses. Tunggu sampai parsing selesai sebelum menghapusnya."})
		return
	}
	if storageKey == "" || filepath.Base(storageKey) != storageKey || storageKey == "." || storageKey == ".." {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lokasi penyimpanan dokumen tidak valid."})
		return
	}
	storageRoot, err := filepath.Abs(h.StorageRoot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lokasi penyimpanan dokumen tidak valid."})
		return
	}
	storagePath, err := filepath.Abs(filepath.Join(storageRoot, storageKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lokasi file dokumen tidak valid."})
		return
	}
	relativePath, err := filepath.Rel(storageRoot, storagePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lokasi file dokumen berada di luar penyimpanan."})
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM extracted_fields WHERE pa_request_id=$1 AND document_id=$2`, requestID, documentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Field hasil ekstraksi dokumen gagal dibersihkan."})
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM checklist_results WHERE pa_request_id=$1`, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hasil checklist lama gagal dibersihkan."})
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM documents WHERE id=$1 AND pa_request_id=$2`, documentID, requestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Metadata dokumen gagal dihapus."})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Penghapusan dokumen gagal disimpan."})
		return
	}
	if err = os.Remove(storagePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("delete document %d storage cleanup failed: %v", documentID, err)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Dokumen dan field hasil ekstraksinya berhasil dihapus."})
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database service unavailable.",
		})
		return
	}

	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	userID := c.GetInt64("user_id")
	role := c.GetString("role")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Periksa keberadaan request dan hak akses sebelum menerima file.
	var createdBy int64
	var requestStatus string
	err := h.DB.QueryRow(ctx, `
		SELECT created_by, status
		FROM pa_requests
		WHERE id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&createdBy, &requestStatus)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "PA request tidak ditemukan.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memeriksa PA request.",
		})
		return
	}

	if role != "staff" || userID != createdBy {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "PA request tidak ditemukan.",
		})
		return
	}
	if requestStatus != "draft" && requestStatus != "needs_information" && requestStatus != "payer_needs_information" {
		c.JSON(http.StatusConflict, gin.H{"error": "Dokumen hanya dapat ditambah saat draft atau ketika Staff sedang melengkapi permintaan informasi."})
		return
	}

	// Batasi keseluruhan request, termasuk multipart overhead.
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		services.MaxDocumentSize+maxMultipartOverhead,
	)

	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Ukuran request terlalu besar. Maksimum file adalah 10 MiB.",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format multipart tidak valid.",
		})
		return
	}

	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File wajib dikirim menggunakan field multipart bernama 'file'.",
		})
		return
	}
	defer file.Close()

	if header.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File kosong.",
		})
		return
	}

	if header.Size > services.MaxDocumentSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "Ukuran file melebihi batas 10 MiB.",
		})
		return
	}

	content, err := io.ReadAll(
		io.LimitReader(file, services.MaxDocumentSize+1),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File tidak dapat dibaca.",
		})
		return
	}

	mimeType, err := services.ValidateDocument(header.Filename, content)
	if err != nil {
		if errors.Is(err, services.ErrFileTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Ukuran file melebihi batas 10 MiB.",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	originalFilename := path.Base(
		strings.ReplaceAll(header.Filename, "\\", "/"),
	)

	if originalFilename == "." ||
		originalFilename == "/" ||
		strings.TrimSpace(originalFilename) == "" ||
		len(originalFilename) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nama file tidak valid atau terlalu panjang.",
		})
		return
	}

	// Gunakan nama acak; jangan gunakan nama pengguna sebagai storage path.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat identitas penyimpanan file.",
		})
		return
	}

	extension := strings.ToLower(filepath.Ext(originalFilename))
	storageKey := hex.EncodeToString(randomBytes) + extension

	if err := os.MkdirAll(h.StorageRoot, 0700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Direktori penyimpanan tidak dapat disiapkan.",
		})
		return
	}

	storagePath := filepath.Join(h.StorageRoot, storageKey)

	if err := writePrivateFile(storagePath, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "File gagal disimpan.",
		})
		return
	}

	// Jika database gagal, hapus file agar tidak menjadi orphan.
	cleanupFile := true
	defer func() {
		if cleanupFile {
			_ = os.Remove(storagePath)
		}
	}()

	var document struct {
		ID            int64     `json:"id"`
		PARequestID   int64     `json:"pa_request_id"`
		OriginalName  string    `json:"original_filename"`
		MIMEType      string    `json:"mime_type"`
		SizeBytes     int64     `json:"size_bytes"`
		ParsingStatus string    `json:"parsing_status"`
		CreatedAt     time.Time `json:"created_at"`
	}

	err = h.DB.QueryRow(ctx, `
		INSERT INTO documents (
			pa_request_id,
			original_filename,
			storage_key,
			mime_type,
			size_bytes,
			parsing_status,
			uploaded_by
		)
		SELECT $1, $2, $3, $4, $5, 'pending', $6
		WHERE EXISTS (
			SELECT 1
			FROM pa_requests
			WHERE id = $1 AND deleted_at IS NULL
		)
		RETURNING id, pa_request_id, original_filename,
		          mime_type, size_bytes, parsing_status, created_at
	`,
		requestID,
		originalFilename,
		storageKey,
		mimeType,
		int64(len(content)),
		userID,
	).Scan(
		&document.ID,
		&document.PARequestID,
		&document.OriginalName,
		&document.MIMEType,
		&document.SizeBytes,
		&document.ParsingStatus,
		&document.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "PA request tidak ditemukan atau sudah dihapus.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Metadata dokumen gagal disimpan.",
		})
		return
	}

	cleanupFile = false

	c.JSON(http.StatusCreated, gin.H{
		"message": "Dokumen berhasil diunggah.",
		"data":    document,
	})
}

func writePrivateFile(destination string, content []byte) error {
	file, err := os.OpenFile(
		destination,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(content)
	closeErr := file.Close()

	if writeErr != nil {
		_ = os.Remove(destination)
		return writeErr
	}

	if closeErr != nil {
		_ = os.Remove(destination)
		return fmt.Errorf("close stored file: %w", closeErr)
	}

	return nil
}
