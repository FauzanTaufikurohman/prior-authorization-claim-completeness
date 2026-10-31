package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WhatsAppDeviceHandler struct {
	DB *pgxpool.Pool
}

type createWhatsAppDevicePairingRequest struct {
	DeviceNumber  string `json:"device_number" binding:"required,max=30"`
	ConfirmCreate bool   `json:"confirm_create"`
}

type whatsappDevicePairing struct {
	ID           int64  `json:"id"`
	DeviceNumber string `json:"device_number"`
	Status       string `json:"status"`
	DeviceName   string `json:"-"`
}

func NewWhatsAppDeviceHandler(db *pgxpool.Pool) *WhatsAppDeviceHandler {
	return &WhatsAppDeviceHandler{DB: db}
}

func (h *WhatsAppDeviceHandler) Current(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	pairing, err := h.loadPairing(ctx, 0, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status koneksi WhatsApp tidak dapat dimuat."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pairing})
}

func (h *WhatsAppDeviceHandler) DeleteCurrent(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if _, err := h.DB.Exec(ctx, `
		DELETE FROM whatsapp_device_pairings
		WHERE staff_user_id = $1
	`, userID); err != nil {
		log.Printf("delete WhatsApp pairing for staff user %d failed: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Catatan pairing WhatsApp tidak dapat dihapus."})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *WhatsAppDeviceHandler) Create(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	var input createWhatsAppDevicePairingRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Masukkan nomor WhatsApp Staff yang valid."})
		return
	}
	if !input.ConfirmCreate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Konfirmasi penambahan device Fonnte diperlukan."})
		return
	}

	deviceNumber, err := services.NormalizeWhatsAppNumber(input.DeviceNumber)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountToken := strings.TrimSpace(os.Getenv("FONNTE_ACCOUNT_TOKEN"))
	webhookURL := strings.TrimSpace(os.Getenv("FONNTE_WEBHOOK_URL"))
	encryptionKey := configuredWhatsAppWebhookSecret()
	if accountToken == "" || len([]byte(encryptionKey)) < 32 || !validWebhookURL(webhookURL) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Pairing belum dikonfigurasi. Isi FONNTE_ACCOUNT_TOKEN dan FONNTE_WEBHOOK_URL dengan URL publik yang memuat secret acak minimal 32 karakter.",
		})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pairing WhatsApp tidak dapat dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pairing WhatsApp tidak dapat dikunci."})
		return
	}

	var pairingID int64
	var currentNumber, encryptedToken, status string
	err = tx.QueryRow(ctx, `
		SELECT id, device_number, encrypted_device_token, status
		FROM whatsapp_device_pairings
		WHERE staff_user_id = $1
		FOR UPDATE
	`, userID).Scan(&pairingID, &currentNumber, &encryptedToken, &status)

	var deviceToken string
	if err == nil {
		if currentNumber != deviceNumber {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Akun ini sudah memiliki device WhatsApp. Hubungi Admin untuk mengganti nomor.",
			})
			return
		}
		deviceToken, err = services.DecryptFonnteToken(encryptedToken, encryptionKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kredensial device WhatsApp tidak dapat dibuka."})
			return
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status pairing WhatsApp tidak dapat diperiksa."})
		return
	} else {
		var existingOwner int64
		err = tx.QueryRow(ctx, `
			SELECT staff_user_id
			FROM whatsapp_device_pairings
			WHERE device_number = $1
		`, deviceNumber).Scan(&existingOwner)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Nomor WhatsApp ini sudah dipasangkan ke akun lain."})
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Nomor WhatsApp tidak dapat diperiksa."})
			return
		}

		deviceName := fmt.Sprintf("PA Staff %d", userID)
		deviceToken, _, err = services.FindFonnteDeviceToken(ctx, accountToken, deviceNumber)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Daftar device Fonnte tidak dapat diperiksa. Periksa FONNTE_ACCOUNT_TOKEN."})
			return
		}
		if deviceToken == "" {
			deviceToken, err = services.AddFonnteDevice(ctx, accountToken, deviceName, deviceNumber)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": "Fonnte tidak dapat menambahkan device. Periksa izin token akun dan kuota device Fonnte.",
				})
				return
			}
		}
		encryptedToken, err = services.EncryptFonnteToken(deviceToken, encryptionKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Fonnte sudah memiliki device ini, tetapi token tidak dapat disimpan. Periksa device melalui Fonnte.",
			})
			return
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO whatsapp_device_pairings (
				staff_user_id, device_number, device_name, encrypted_device_token
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id, status
		`, userID, deviceNumber, deviceName, encryptedToken).Scan(&pairingID, &status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Device sudah tercatat di Fonnte, tetapi pairing tidak dapat disimpan. Periksa device melalui Fonnte.",
			})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Device sudah tercatat di Fonnte, tetapi pairing tidak dapat disimpan. Periksa device melalui Fonnte.",
		})
		return
	}

	qrCode, err := h.refreshQRCode(ctx, pairingID, userID, deviceToken)
	if err != nil {
		log.Printf("load QR for WhatsApp pairing %d failed: %v", pairingID, err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Device sudah tercatat, tetapi Fonnte gagal menyiapkan QR. Periksa log backend dan pastikan paket Fonnte mengizinkan device tambahan tersambung.",
			"data":  gin.H{"id": pairingID, "device_number": deviceNumber, "status": status},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id": pairingID, "device_number": deviceNumber, "status": status, "qr_code": qrCode,
	}})
}

func (h *WhatsAppDeviceHandler) RefreshQRCode(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	pairing, encryptedToken, err := h.loadPairingWithToken(ctx, 0, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Belum ada device WhatsApp untuk akun ini."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data pairing WhatsApp tidak dapat dimuat."})
		return
	}
	deviceToken, err := services.DecryptFonnteToken(encryptedToken, configuredWhatsAppWebhookSecret())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Kredensial device tidak dapat dibuka. Periksa secret pada FONNTE_WEBHOOK_URL."})
		return
	}
	qrCode, err := h.refreshQRCode(ctx, pairing.ID, userID, deviceToken)
	if err != nil {
		log.Printf("refresh QR for WhatsApp pairing %d failed: %v", pairing.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "QR Fonnte belum dapat dimuat. Periksa log backend, status device, dan batas device pada paket Fonnte.",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id": pairing.ID, "device_number": pairing.DeviceNumber, "status": pairing.Status, "qr_code": qrCode,
	}})
}

func (h *WhatsAppDeviceHandler) ConfigureWebhook(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}

	webhookURL := strings.TrimSpace(os.Getenv("FONNTE_WEBHOOK_URL"))
	if !validWebhookURL(webhookURL) || len([]byte(configuredWhatsAppWebhookSecret())) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "FONNTE_WEBHOOK_URL belum valid atau secret webhook kurang dari 32 karakter.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	pairing, encryptedToken, err := h.loadPairingWithToken(ctx, 0, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pairing WhatsApp tidak ditemukan."})
		return
	}
	if err != nil {
		log.Printf("load WhatsApp pairing for webhook configuration for staff %d failed: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pairing WhatsApp tidak dapat dimuat."})
		return
	}
	deviceToken, err := services.DecryptFonnteToken(encryptedToken, configuredWhatsAppWebhookSecret())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Kredensial device tidak dapat dibuka. Pastikan secret webhook tidak berubah.",
		})
		return
	}
	if err := services.ConfigureFonnteDevice(
		ctx, deviceToken, pairing.DeviceName, pairing.DeviceNumber, webhookURL,
	); err != nil {
		log.Printf("configure WhatsApp webhook for pairing %d failed: %v", pairing.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Fonnte gagal memperbarui webhook. Periksa koneksi dan token device.",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

func (h *WhatsAppDeviceHandler) Status(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi pengguna tidak valid."})
		return
	}
	pairingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || pairingID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pairing WhatsApp tidak valid."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	pairing, encryptedToken, err := h.loadPairingWithToken(ctx, pairingID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pairing WhatsApp tidak ditemukan."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status pairing WhatsApp tidak dapat dimuat."})
		return
	}
	deviceToken, err := services.DecryptFonnteToken(encryptedToken, configuredWhatsAppWebhookSecret())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Kredensial device tidak dapat dibuka. Periksa secret pada FONNTE_WEBHOOK_URL."})
		return
	}
	profile, err := services.GetFonnteDeviceProfile(ctx, deviceToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Status device belum dapat diperiksa. Coba muat ulang beberapa saat lagi."})
		return
	}

	status := "pending"
	switch {
	case strings.EqualFold(profile.DeviceStatus, "connect"):
		status = "connected"
	case strings.EqualFold(profile.DeviceStatus, "disconnect"):
		if pairing.Status == "connected" || pairing.Status == "disconnected" {
			status = "disconnected"
		}
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": "Fonnte mengembalikan status device yang tidak dikenal."})
		return
	}
	if _, err := h.DB.Exec(ctx, `
		UPDATE whatsapp_device_pairings
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND staff_user_id = $3
	`, status, pairing.ID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status device tidak dapat disimpan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id": pairing.ID, "device_number": pairing.DeviceNumber, "status": status,
	}})
}

func (h *WhatsAppDeviceHandler) refreshQRCode(
	ctx context.Context,
	pairingID, userID int64,
	deviceToken string,
) (string, error) {
	var deviceNumber, deviceName string
	if err := h.DB.QueryRow(ctx, `
		SELECT device_number, device_name
		FROM whatsapp_device_pairings
		WHERE id = $1 AND staff_user_id = $2
	`, pairingID, userID).Scan(&deviceNumber, &deviceName); err != nil {
		return "", fmt.Errorf("load pairing before QR refresh: %w", err)
	}
	webhookURL := strings.TrimSpace(os.Getenv("FONNTE_WEBHOOK_URL"))
	if !validWebhookURL(webhookURL) {
		return "", errors.New("FONNTE_WEBHOOK_URL is not configured")
	}
	if err := services.ConfigureFonnteDevice(ctx, deviceToken, deviceName, deviceNumber, webhookURL); err != nil {
		return "", fmt.Errorf("configure WhatsApp webhook: %w", err)
	}
	return services.GetFonnteDeviceQRCode(ctx, deviceToken)
}

func (h *WhatsAppDeviceHandler) loadPairing(
	ctx context.Context,
	pairingID, userID int64,
) (whatsappDevicePairing, error) {
	pairing, _, err := h.loadPairingWithToken(ctx, pairingID, userID)
	return pairing, err
}

func (h *WhatsAppDeviceHandler) loadPairingWithToken(
	ctx context.Context,
	pairingID, userID int64,
) (whatsappDevicePairing, string, error) {
	var pairing whatsappDevicePairing
	var encryptedToken string
	query := `
		SELECT id, device_number, status, encrypted_device_token, device_name
		FROM whatsapp_device_pairings
		WHERE staff_user_id = $1
	`
	args := []any{userID}
	if pairingID > 0 {
		query += ` AND id = $2`
		args = append(args, pairingID)
	}
	err := h.DB.QueryRow(ctx, query, args...).Scan(
		&pairing.ID,
		&pairing.DeviceNumber,
		&pairing.Status,
		&encryptedToken,
		&pairing.DeviceName,
	)
	return pairing, encryptedToken, err
}

func authenticatedUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok && userID > 0
}
