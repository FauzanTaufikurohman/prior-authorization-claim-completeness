package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	whatsAppIntakeAwaitingConsent      = "awaiting_consent"
	whatsAppIntakeCollecting           = "collecting"
	whatsAppIntakeAwaitingConfirmation = "awaiting_confirmation"
	whatsAppIntakeCompleted            = "completed"
	whatsAppIntakeCancelled            = "cancelled"
)

type WhatsAppIntakeHandler struct {
	DB *pgxpool.Pool
}

type whatsAppWebhookID string

func (id *whatsAppWebhookID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*id = ""
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*id = whatsAppWebhookID(value)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	*id = whatsAppWebhookID(number.String())
	return nil
}

type whatsAppWebhookMessage struct {
	Device  string            `json:"device"`
	Sender  string            `json:"sender"`
	Message string            `json:"message"`
	Text    string            `json:"text"`
	InboxID whatsAppWebhookID `json:"inboxid"`
}

type whatsAppIntakeData struct {
	PayerName            string `json:"payer_name"`
	ServiceType          string `json:"service_type"`
	ServiceTypeSuggested bool   `json:"service_type_suggested"`
	OrderingProviderNPI  string `json:"ordering_provider_npi"`
	ServicingProviderNPI string `json:"servicing_provider_npi"`
	PatientReference     string `json:"patient_reference"`
	Symptoms             string `json:"symptoms"`
	Diagnosis            string `json:"diagnosis"`
}

type whatsAppIntakeActivity struct {
	ID              int64      `json:"id"`
	SenderMasked    string     `json:"sender_masked"`
	Status          string     `json:"status"`
	LastAIStatus    string     `json:"last_ai_status"`
	LastAIErrorCode string     `json:"last_ai_error_code"`
	LastMessageAt   *time.Time `json:"last_message_at"`
	LastAIAttemptAt *time.Time `json:"last_ai_attempt_at"`
	PARequestNumber string     `json:"pa_request_number"`
}

func NewWhatsAppIntakeHandler(db *pgxpool.Pool) *WhatsAppIntakeHandler {
	return &WhatsAppIntakeHandler{DB: db}
}

func (h *WhatsAppIntakeHandler) Activity(c *gin.Context) {
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
	query := `
		SELECT i.id, i.sender_number, i.status,
		       COALESCE(i.last_ai_status, ''), COALESCE(i.last_ai_error_code, ''),
		       i.last_message_at,
		       i.last_ai_attempt_at, COALESCE(p.request_number, '')
		FROM whatsapp_pa_intakes i
		LEFT JOIN pa_requests p ON p.id = i.pa_request_id
	`
	args := []any{}
	if c.GetString("role") != "admin" {
		query += ` WHERE i.staff_user_id = $1`
		args = append(args, userID)
	}
	query += ` ORDER BY i.updated_at DESC, i.id DESC LIMIT 50`

	rows, err := h.DB.Query(ctx, query, args...)
	if err != nil {
		log.Printf("list WhatsApp intake activity for user %d failed: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivitas chatbot tidak dapat dimuat."})
		return
	}
	defer rows.Close()

	activities := make([]whatsAppIntakeActivity, 0)
	for rows.Next() {
		var activity whatsAppIntakeActivity
		var senderNumber string
		if err := rows.Scan(
			&activity.ID,
			&senderNumber,
			&activity.Status,
			&activity.LastAIStatus,
			&activity.LastAIErrorCode,
			&activity.LastMessageAt,
			&activity.LastAIAttemptAt,
			&activity.PARequestNumber,
		); err != nil {
			log.Printf("scan WhatsApp intake activity for user %d failed: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivitas chatbot tidak dapat dibaca."})
			return
		}
		activity.SenderMasked = maskWhatsAppSender(senderNumber)
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read WhatsApp intake activity for user %d failed: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aktivitas chatbot tidak dapat dibaca."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": activities})
}

func maskWhatsAppSender(number string) string {
	if len(number) <= 4 {
		return "****"
	}
	return "****" + number[len(number)-4:]
}

func whatsAppWebhookDecodeDiagnostic(err error) (reason, field, expectedType string, offset int64) {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return "type_mismatch", typeError.Field, typeError.Type.String(), typeError.Offset
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return "syntax_error", "", "", syntaxError.Offset
	}

	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return "body_too_large", "", "", maxBytesError.Limit
	}

	return "decode_error", "", "", 0
}

func (h *WhatsAppIntakeHandler) Webhook(c *gin.Context) {
	if h.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable."})
		return
	}

	webhookURL := strings.TrimSpace(os.Getenv("FONNTE_WEBHOOK_URL"))
	frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_PUBLIC_URL"))
	secret := configuredWhatsAppWebhookSecret()
	providedSecret := c.Query("secret")
	if !validWebhookURL(webhookURL) || len([]byte(secret)) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "FONNTE_WEBHOOK_URL harus berisi URL publik webhook dan secret acak minimal 32 karakter pada parameter secret.",
		})
		return
	}
	if subtle.ConstantTimeCompare([]byte(secret), []byte(providedSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Secret webhook tidak valid."})
		return
	}
	if _, err := buildWhatsAppUploadURL(frontendURL, "configuration-check"); err != nil {
		log.Printf("WhatsApp webhook unavailable: reason=invalid_frontend_public_url")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "FRONTEND_PUBLIC_URL harus berisi URL HTTP atau HTTPS aplikasi frontend.",
		})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	var input whatsAppWebhookMessage
	if err := c.ShouldBindJSON(&input); err != nil {
		reason, field, expectedType, offset := whatsAppWebhookDecodeDiagnostic(err)
		log.Printf(
			"WhatsApp webhook rejected: reason=invalid_json decode_reason=%s field=%q expected_type=%q offset=%d content_type=%q content_length=%d",
			reason,
			field,
			expectedType,
			offset,
			c.GetHeader("Content-Type"),
			c.Request.ContentLength,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format pesan webhook WhatsApp tidak valid."})
		return
	}
	if strings.TrimSpace(input.Device) == "" || strings.TrimSpace(input.Sender) == "" {
		log.Printf(
			"WhatsApp webhook rejected: reason=missing_fields device_present=%t sender_present=%t message_present=%t text_present=%t",
			strings.TrimSpace(input.Device) != "",
			strings.TrimSpace(input.Sender) != "",
			strings.TrimSpace(input.Message) != "",
			strings.TrimSpace(input.Text) != "",
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nomor device dan pengirim wajib tersedia."})
		return
	}
	if strings.TrimSpace(input.Message) == "" {
		input.Message = input.Text
	}
	if !utf8.ValidString(input.Message) {
		log.Printf("WhatsApp webhook rejected: reason=invalid_message_encoding")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan WhatsApp tidak valid."})
		return
	}

	deviceNumber, err := services.NormalizeWhatsAppNumber(input.Device)
	if err != nil {
		log.Printf("WhatsApp webhook rejected: reason=invalid_device_number")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nomor device WhatsApp tidak valid."})
		return
	}
	senderNumber, err := services.NormalizeWhatsAppNumber(input.Sender)
	if err != nil {
		log.Printf("WhatsApp webhook rejected: reason=unsupported_sender_number")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pesan grup atau nomor pengirim tidak didukung."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 75*time.Second)
	defer cancel()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Percakapan WhatsApp tidak dapat dimulai."})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var staffUserID int64
	var encryptedDeviceToken string
	err = tx.QueryRow(ctx, `
		SELECT p.staff_user_id, p.encrypted_device_token
		FROM whatsapp_device_pairings p
		JOIN users u ON u.id = p.staff_user_id AND u.is_active = TRUE AND u.role = 'staff'
		WHERE p.device_number = $1
	`, deviceNumber).Scan(&staffUserID, &encryptedDeviceToken)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device WhatsApp belum dipasangkan ke akun Staff."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pemilik device WhatsApp tidak dapat diperiksa."})
		return
	}

	deviceToken, err := services.DecryptFonnteToken(encryptedDeviceToken, secret)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Token device tidak dapat dibuka. Pastikan secret pada FONNTE_WEBHOOK_URL tidak berubah sejak pairing.",
		})
		return
	}

	var intakeID int64
	var status string
	var storedData []byte
	var encryptedCompletionMessage string
	err = tx.QueryRow(ctx, `
		SELECT id, status, collected_data, COALESCE(encrypted_completion_message, '')
		FROM whatsapp_pa_intakes
		WHERE device_number = $1 AND sender_number = $2
		ORDER BY id DESC
		LIMIT 1
		FOR UPDATE
	`, deviceNumber, senderNumber).Scan(&intakeID, &status, &storedData, &encryptedCompletionMessage)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Percakapan WhatsApp tidak dapat dimuat."})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) ||
		((status == whatsAppIntakeCompleted || status == whatsAppIntakeCancelled) && isWhatsAppIntakeNewRequest(input.Message)) {
		_, err = tx.Exec(ctx, `
			INSERT INTO whatsapp_pa_intakes (device_number, sender_number, staff_user_id, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (device_number, sender_number)
			WHERE status IN ('awaiting_consent', 'collecting', 'awaiting_confirmation')
			DO NOTHING
		`, deviceNumber, senderNumber, staffUserID, whatsAppIntakeAwaitingConsent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Percakapan WhatsApp tidak dapat disimpan."})
			return
		}

		err = tx.QueryRow(ctx, `
			SELECT id, status, collected_data, COALESCE(encrypted_completion_message, '')
			FROM whatsapp_pa_intakes
			WHERE device_number = $1 AND sender_number = $2
			  AND status IN ('awaiting_consent', 'collecting', 'awaiting_confirmation')
			ORDER BY id DESC
			LIMIT 1
			FOR UPDATE
		`, deviceNumber, senderNumber).Scan(&intakeID, &status, &storedData, &encryptedCompletionMessage)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Percakapan WhatsApp tidak dapat dimuat."})
			return
		}
	}

	var data whatsAppIntakeData
	if len(storedData) > 0 {
		if err := json.Unmarshal(storedData, &data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Data percakapan WhatsApp tidak dapat dibaca."})
			return
		}
	}

	aiStatus := ""
	aiErrorCode := ""
	reply, err := h.processMessage(
		ctx, tx, intakeID, staffUserID, senderNumber, status, data,
		input.Message, secret, frontendURL, &aiStatus, &aiErrorCode,
	)
	if err != nil {
		log.Printf("process WhatsApp PA intake for staff %d failed: %v", staffUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pesan WhatsApp gagal diproses."})
		return
	}
	_, err = tx.Exec(ctx, `
		UPDATE whatsapp_pa_intakes
		SET last_message_at = NOW(),
		    last_ai_status = CASE WHEN $1 IN ('processed', 'failed') THEN $1 ELSE last_ai_status END,
		    last_ai_attempt_at = CASE WHEN $1 IN ('processed', 'failed') THEN NOW() ELSE last_ai_attempt_at END,
		    last_ai_error_code = CASE
		        WHEN $1 = 'failed' THEN NULLIF($2, '')
		        WHEN $1 = 'processed' THEN NULL
		        ELSE last_ai_error_code
		    END,
		    updated_at = NOW()
		WHERE id = $3
	`, aiStatus, aiErrorCode, intakeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status pemrosesan pesan tidak dapat disimpan."})
		return
	}
	if status == whatsAppIntakeCompleted && !isWhatsAppIntakeHelp(input.Message) {
		reply, err = services.DecryptFonnteToken(encryptedCompletionMessage, secret)
		if err != nil {
			log.Printf("decrypt WhatsApp intake completion reply for staff %d failed: %v", staffUserID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Link upload tidak dapat dipulihkan."})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Percakapan WhatsApp gagal disimpan."})
		return
	}
	if err := services.SendFonnteReply(ctx, deviceToken, senderNumber, string(input.InboxID), reply); err != nil {
		log.Printf("send WhatsApp PA intake reply for staff %d failed: %v", staffUserID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Balasan WhatsApp gagal dikirim. Fonnte dapat mencoba kembali webhook ini."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

func (h *WhatsAppIntakeHandler) processMessage(
	ctx context.Context,
	tx pgx.Tx,
	intakeID, staffUserID int64,
	senderNumber, status string,
	data whatsAppIntakeData,
	message, secret, frontendURL string,
	aiStatus, aiErrorCode *string,
) (string, error) {
	normalizedMessage := strings.ToUpper(strings.TrimSpace(message))
	if isWhatsAppIntakeHelp(normalizedMessage) {
		return whatsAppIntakeHelpMessage(), nil
	}
	if status == whatsAppIntakeCompleted {
		return "Permohonan sebelumnya sudah selesai. Balas MULAI atau BUAT PA BARU untuk membuat permohonan lain.\n\n" + whatsAppIntakeHelpMessage(), nil
	}
	if status == whatsAppIntakeCancelled {
		if !isWhatsAppIntakeNewRequest(normalizedMessage) {
			return "Percakapan sebelumnya dibatalkan. Balas MULAI atau BUAT PA BARU untuk memulai lagi.\n\n" + whatsAppIntakeHelpMessage(), nil
		}
		data = whatsAppIntakeData{}
		status = whatsAppIntakeAwaitingConsent
		if err := updateWhatsAppIntake(ctx, tx, intakeID, status, data); err != nil {
			return "", err
		}
	}
	if (status == whatsAppIntakeCollecting || status == whatsAppIntakeAwaitingConfirmation) &&
		data.PatientReference == "" {
		patientReference, err := generateWhatsAppSyntheticPatientReference()
		if err != nil {
			return "", fmt.Errorf("generate synthetic WhatsApp patient reference: %w", err)
		}
		data.PatientReference = patientReference
		if err := updateWhatsAppIntake(ctx, tx, intakeID, status, data); err != nil {
			return "", err
		}
	}
	if status == whatsAppIntakeCollecting && data.ServiceType == "" && normalizedMessage == "LANJUTKAN" {
		data.ServiceType = "Perlu ditentukan Staff"
		data.ServiceTypeSuggested = false
		if err := updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCollecting, data); err != nil {
			return "", err
		}
	}
	if normalizedMessage == "BATAL" {
		_, err := tx.Exec(ctx, `
			UPDATE whatsapp_pa_intakes
			SET status = $1, collected_data = '{}'::jsonb, updated_at = NOW()
			WHERE id = $2
		`, whatsAppIntakeCancelled, intakeID)
		return "Pengumpulan informasi dibatalkan. Balas MULAI jika ingin memulai lagi.", err
	}
	if status == whatsAppIntakeAwaitingConfirmation && isWhatsAppIntakeRejection(normalizedMessage) {
		if err := updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCancelled, whatsAppIntakeData{}); err != nil {
			return "", err
		}
		return "Baik, draft PA tidak dibuat. Jika ingin mengulang pengumpulan, balas MULAI.", nil
	}

	switch status {
	case whatsAppIntakeAwaitingConsent:
		if !isWhatsAppIntakeConsent(normalizedMessage) {
			return whatsAppIntakeHelpMessage(), nil
		}
		if data.PatientReference == "" {
			patientReference, err := generateWhatsAppSyntheticPatientReference()
			if err != nil {
				return "", fmt.Errorf("generate synthetic WhatsApp patient reference: %w", err)
			}
			data.PatientReference = patientReference
		}
		return "Baik, kita mulai. Saya akan mengumpulkan informasi untuk draft PA. " + nextWhatsAppIntakeQuestion(data), updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCollecting, data)
	case whatsAppIntakeCollecting, whatsAppIntakeAwaitingConfirmation:
		if status == whatsAppIntakeAwaitingConfirmation && isWhatsAppIntakeConfirmation(normalizedMessage) {
			return createWhatsAppPADraft(ctx, tx, intakeID, staffUserID, data, secret, frontendURL)
		}
		if len([]rune(strings.TrimSpace(message))) > services.MaxWhatsAppIntakeMessageCharacters {
			return "Pesan terlalu panjang. Kirim jawaban singkat untuk satu atau beberapa informasi PA.", nil
		}

		fields, err := services.NewOpenAIExtractorFromEnv().ExtractWhatsAppIntakeFields(ctx, message)
		if err != nil {
			*aiStatus = "failed"
			*aiErrorCode = whatsAppAIErrorCode(err)
			log.Printf("extract WhatsApp PA intake fields for staff %d failed: %v", staffUserID, err)
			return "AI belum dapat memproses jawaban ini. Jawaban tidak disimpan. Coba kirim ulang atau lanjutkan pengisian PA bersama Staff.", nil
		}
		*aiStatus = "processed"
		*aiErrorCode = ""
		if err := mergeWhatsAppIntakeFields(&data, fields); err != nil {
			return err.Error(), updateWhatsAppIntake(ctx, tx, intakeID, status, data)
		}
		if missingWhatsAppIntakeQuestion(data) != "" {
			return nextWhatsAppIntakeQuestion(data), updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCollecting, data)
		}
		if question, err := missingWhatsAppProviderQuestion(ctx, tx, data); err != nil {
			return "", fmt.Errorf("check WhatsApp PA provider requirements: %w", err)
		} else if question != "" {
			return question, updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCollecting, data)
		}

		reply := formatWhatsAppIntakeConfirmation(data)
		return reply, updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeAwaitingConfirmation, data)
	default:
		return "", fmt.Errorf("unknown WhatsApp intake status %q", status)
	}
}

func updateWhatsAppIntake(
	ctx context.Context,
	tx pgx.Tx,
	intakeID int64,
	status string,
	data whatsAppIntakeData,
) error {
	encodedData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode WhatsApp intake fields: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE whatsapp_pa_intakes
		SET status = $1, collected_data = $2::jsonb, updated_at = NOW()
		WHERE id = $3
	`, status, string(encodedData), intakeID); err != nil {
		return fmt.Errorf("save WhatsApp intake state: %w", err)
	}
	return nil
}

func mergeWhatsAppIntakeFields(data *whatsAppIntakeData, fields services.WhatsAppIntakeFields) error {
	if fields.ServiceType != nil && strings.TrimSpace(*fields.ServiceType) != "" {
		value := strings.TrimSpace(*fields.ServiceType)
		if len([]rune(value)) > 100 {
			return errors.New("Nilai layanan terlalu panjang. Kirim versi yang lebih singkat.")
		}
		data.ServiceType = value
		data.ServiceTypeSuggested = false
	} else if (data.ServiceType == "" || data.ServiceTypeSuggested) && fields.SuggestedServiceType != nil {
		value := strings.TrimSpace(*fields.SuggestedServiceType)
		if len([]rune(value)) > 100 {
			return errors.New("Usulan layanan terlalu panjang.")
		}
		if value != "" {
			data.ServiceType = value
			data.ServiceTypeSuggested = true
		}
	}

	for _, field := range []struct {
		name  string
		value *string
		dest  *string
		max   int
	}{
		{name: "payer", value: fields.PayerName, dest: &data.PayerName, max: 150},
		{name: "NPI ordering provider", value: fields.OrderingProviderNPI, dest: &data.OrderingProviderNPI, max: 10},
		{name: "NPI servicing provider", value: fields.ServicingProviderNPI, dest: &data.ServicingProviderNPI, max: 10},
		{name: "keluhan", value: fields.Symptoms, dest: &data.Symptoms, max: 2000},
		{name: "diagnosis", value: fields.Diagnosis, dest: &data.Diagnosis, max: 2000},
	} {
		if field.value == nil {
			continue
		}
		value := strings.TrimSpace(*field.value)
		if len([]rune(value)) > field.max {
			return fmt.Errorf("Nilai %s terlalu panjang. Kirim versi yang lebih singkat.", field.name)
		}
		if strings.Contains(field.name, "NPI") && value != "" && !validOptionalNPI(value) {
			return fmt.Errorf("NPI %s harus terdiri dari 10 digit.", strings.TrimPrefix(field.name, "NPI "))
		}
		if value != "" {
			*field.dest = value
		}
	}
	return nil
}

func isWhatsAppIntakeConsent(message string) bool {
	return message == "SETUJU" || message == "SAYA SETUJU"
}

func isWhatsAppIntakeHelp(message string) bool {
	switch strings.Trim(strings.ToUpper(strings.TrimSpace(message)), ".,!;:") {
	case "BANTUAN", "MENU", "HELP", "?":
		return true
	default:
		return false
	}
}

func whatsAppIntakeHelpMessage() string {
	return "Untuk membuat permohonan PA, balas SETUJU. Jawaban keluhan akan diproses AI dan disimpan sebagai draft yang dapat diperiksa Staff. Ceritakan payer/asuransi, keluhan atau gejala, dan diagnosis atau layanan dari dokter jika sudah tahu. AI dapat mengusulkan layanan untuk diperiksa Staff, bukan menentukan keputusan medis. Saya membuat ID pasien sintetis, jadi Anda tidak perlu mengirim nomor rekam medis. Balas BATAL kapan saja untuk berhenti.\n\nUntuk permohonan baru setelah selesai, balas MULAI atau BUAT PA BARU. Untuk melihat petunjuk ini lagi, balas BANTUAN. Jangan kirim nama lengkap, NIK, alamat, tanggal lahir, atau nomor rekam medis. Staff harus memeriksa draft dan usulan AI sebelum PA diproses lebih lanjut."
}

func isWhatsAppIntakeNewRequest(message string) bool {
	switch strings.Trim(strings.ToUpper(strings.TrimSpace(message)), ".,!?;:") {
	case "MULAI", "MULAI REQUEST BARU", "BUAT PA BARU", "PA BARU":
		return true
	default:
		return false
	}
}

func isWhatsAppIntakeConfirmation(message string) bool {
	switch strings.Trim(strings.ToUpper(strings.TrimSpace(message)), ".,!?;:") {
	case "YA", "IYA", "SETUJU", "SAYA SETUJU", "OK", "OKE",
		"BUAT PA", "YA BUAT PA", "YA, BUAT PA",
		"KONFIRMASI", "KONFIRMASI BUAT PA":
		return true
	default:
		return false
	}
}

func isWhatsAppIntakeRejection(message string) bool {
	switch strings.Trim(strings.ToUpper(strings.TrimSpace(message)), ".,!?;:") {
	case "TIDAK", "TIDAK SETUJU", "TIDAK JADI", "JANGAN", "JANGAN BUAT PA":
		return true
	default:
		return false
	}
}

func whatsAppAIErrorCode(err error) string {
	if errors.Is(err, services.ErrOpenAIKeyMissing) {
		return "missing_api_key"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var apiError *services.OpenAIAPIError
	if errors.As(err, &apiError) {
		switch {
		case apiError.StatusCode == http.StatusUnauthorized || apiError.StatusCode == http.StatusForbidden:
			return "authentication"
		case apiError.StatusCode == http.StatusNotFound:
			return "model_not_found"
		case apiError.StatusCode == http.StatusTooManyRequests:
			return "rate_limited"
		case apiError.StatusCode == http.StatusBadRequest:
			return "request_rejected"
		case apiError.StatusCode >= http.StatusInternalServerError:
			return "provider_error"
		default:
			return "request_failed"
		}
	}

	var requestError *url.Error
	if errors.As(err, &requestError) {
		if requestError.Timeout() || errors.Is(requestError.Err, context.DeadlineExceeded) {
			return "timeout"
		}
		return "connection_error"
	}
	return "invalid_response"
}

func missingWhatsAppIntakeQuestion(data whatsAppIntakeData) string {
	switch {
	case data.PayerName == "":
		return "Nama payer atau asuransi apa yang digunakan?"
	case data.ServiceType == "":
		if data.Diagnosis == "" && data.Symptoms == "" {
			return "Ceritakan keluhan atau gejala. Jika sudah mengetahui diagnosis, sebutkan juga."
		}
		return "Apakah dokter sudah menyebut pemeriksaan, tindakan, atau obat yang diminta? Jika belum, balas LANJUTKAN agar Staff menentukan layanan setelah meninjau draft."
	case data.Diagnosis == "" && data.Symptoms == "":
		return "Ceritakan keluhan atau gejala. Jika sudah mengetahui diagnosis, sebutkan juga."
	case data.PatientReference == "":
		return "ID referensi sintetis belum dibuat. Balas MULAI untuk memulai ulang percakapan."
	default:
		return ""
	}
}

func nextWhatsAppIntakeQuestion(data whatsAppIntakeData) string {
	if question := missingWhatsAppIntakeQuestion(data); question != "" {
		return question
	}
	return formatWhatsAppIntakeConfirmation(data)
}

func formatWhatsAppIntakeConfirmation(data whatsAppIntakeData) string {
	serviceLabel := "Layanan"
	serviceNote := "Staff tetap perlu memeriksa layanan sebelum melanjutkan PA."
	if data.ServiceTypeSuggested {
		serviceLabel = "Usulan layanan AI, wajib diperiksa Staff"
		serviceNote = "Usulan AI bukan keputusan medis dan wajib diperiksa Staff."
	} else if data.ServiceType == "Perlu ditentukan Staff" {
		serviceLabel = "Layanan belum ditentukan"
		serviceNote = "Staff perlu menentukan layanan saat memeriksa draft."
	}
	clinicalDetails := data.Diagnosis
	if data.Symptoms != "" {
		if clinicalDetails != "" {
			clinicalDetails += "\n"
		}
		clinicalDetails += "Keluhan yang dilaporkan: " + data.Symptoms
	}
	if clinicalDetails == "" {
		clinicalDetails = "Belum diisi"
	}
	return fmt.Sprintf(
		"Periksa data untuk draft PA:\nPayer: %s\n%s: %s\nNPI ordering provider: %s\nNPI servicing provider: %s\nReferensi pasien: %s (ID sintetis, Staff perlu mengganti atau memverifikasinya)\nDiagnosis dan keluhan: %s\n\n%s Balas YA untuk membuat draft agar diperiksa Staff, atau TIDAK untuk membatalkan. Jika ada koreksi, kirim bagian yang perlu diperbaiki.",
		data.PayerName,
		serviceLabel,
		data.ServiceType,
		whatsAppIntakeValueOrUnfilled(data.OrderingProviderNPI),
		whatsAppIntakeValueOrUnfilled(data.ServicingProviderNPI),
		data.PatientReference,
		clinicalDetails,
		serviceNote,
	)
}

func formatWhatsAppPADiagnosis(data whatsAppIntakeData) string {
	if data.Diagnosis == "" {
		return "Keluhan yang dilaporkan: " + data.Symptoms
	}
	if data.Symptoms == "" {
		return data.Diagnosis
	}
	return data.Diagnosis + "\nKeluhan yang dilaporkan: " + data.Symptoms
}

func whatsAppIntakeValueOrUnfilled(value string) string {
	if value == "" {
		return "Belum diisi"
	}
	return value
}

func missingWhatsAppProviderQuestion(ctx context.Context, tx pgx.Tx, data whatsAppIntakeData) (string, error) {
	checklistVersionID, err := findActiveChecklistVersion(ctx, tx, data.PayerName, data.ServiceType)
	if err != nil {
		return "", err
	}
	if checklistVersionID == nil {
		return "", nil
	}

	var orderingRequired, servicingRequired bool
	if err := tx.QueryRow(ctx, `
		SELECT
			COALESCE(bool_or(lower(replace(required_field_key, '-', '_')) IN ('provider_npi', 'ordering_provider_npi')), FALSE),
			COALESCE(bool_or(lower(replace(required_field_key, '-', '_')) = 'servicing_provider_npi'), FALSE)
		FROM checklist_rules
		WHERE checklist_version_id = $1 AND is_required = TRUE AND required_field_key IS NOT NULL
	`, *checklistVersionID).Scan(&orderingRequired, &servicingRequired); err != nil {
		return "", err
	}

	missing := make([]string, 0, 2)
	if orderingRequired && data.OrderingProviderNPI == "" {
		missing = append(missing, "ordering provider")
	}
	if servicingRequired && data.ServicingProviderNPI == "" {
		missing = append(missing, "servicing provider")
	}
	if len(missing) == 0 {
		return "", nil
	}
	if len(missing) == 2 {
		return "Checklist aktif mewajibkan NPI ordering provider dan servicing provider. Kirim kedua NPI tersebut, masing-masing 10 digit.", nil
	}
	return fmt.Sprintf("Checklist aktif mewajibkan NPI %s. Kirim NPI tersebut dalam 10 digit.", missing[0]), nil
}

func createWhatsAppPADraft(
	ctx context.Context,
	tx pgx.Tx,
	intakeID, staffUserID int64,
	data whatsAppIntakeData,
	secret, frontendURL string,
) (string, error) {
	if missingWhatsAppIntakeQuestion(data) != "" {
		return nextWhatsAppIntakeQuestion(data), nil
	}
	if question, err := missingWhatsAppProviderQuestion(ctx, tx, data); err != nil {
		return "", fmt.Errorf("check WhatsApp PA provider requirements: %w", err)
	} else if question != "" {
		if err := updateWhatsAppIntake(ctx, tx, intakeID, whatsAppIntakeCollecting, data); err != nil {
			return "", err
		}
		return question, nil
	}
	requestNumber, err := services.GenerateRequestNumber(time.Now())
	if err != nil {
		return "", fmt.Errorf("generate PA request number: %w", err)
	}
	checklistVersionID, err := findActiveChecklistVersion(ctx, tx, data.PayerName, data.ServiceType)
	if err != nil {
		return "", fmt.Errorf("find active checklist for WhatsApp intake: %w", err)
	}
	uploadToken, err := generatePublicUploadToken()
	if err != nil {
		return "", fmt.Errorf("generate public upload token: %w", err)
	}
	uploadURL, err := buildWhatsAppUploadURL(frontendURL, uploadToken)
	if err != nil {
		return "", fmt.Errorf("build public upload URL: %w", err)
	}

	var requestID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO pa_requests (
			request_number, created_by, payer_name, service_type,
			provider_npi, ordering_provider_npi, servicing_provider_npi,
			patient_reference, diagnosis, status, checklist_version_id,
			public_upload_token_hash
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6::varchar, ''), NULLIF($7::varchar, ''),
			$8, $9, 'draft', $10, $11)
		RETURNING id
	`, requestNumber, staffUserID, data.PayerName, data.ServiceType,
		data.OrderingProviderNPI, data.OrderingProviderNPI, data.ServicingProviderNPI,
		data.PatientReference, formatWhatsAppPADiagnosis(data), checklistVersionID,
		hashPublicUploadToken(uploadToken)).Scan(&requestID)
	if err != nil {
		return "", fmt.Errorf("create PA draft from WhatsApp: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pa_request_status_history (pa_request_id, actor_id, from_status, to_status, action, note)
		VALUES ($1, $2, NULL, 'draft', 'created_via_whatsapp', 'Draft dibuat setelah konfirmasi melalui WhatsApp.')
	`, requestID, staffUserID); err != nil {
		return "", fmt.Errorf("record WhatsApp PA draft creation: %w", err)
	}

	reply := fmt.Sprintf(
		"Draft PA %s sudah dibuat siapkan dokumen yang dibutuhkan. Unduh dari aplikasi BPJS (contoh).Lalu Unggah dokumen melalui tautan ini: %s\nUntuk membuat permohonan PA baru, balas MULAI.",
		requestNumber,
		uploadURL,
	)
	encryptedReply, err := services.EncryptFonnteToken(reply, secret)
	if err != nil {
		return "", fmt.Errorf("encrypt WhatsApp intake completion message: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE whatsapp_pa_intakes
		SET status = $1, pa_request_id = $2, encrypted_completion_message = $3, updated_at = NOW()
		WHERE id = $4
	`, whatsAppIntakeCompleted, requestID, encryptedReply, intakeID); err != nil {
		return "", fmt.Errorf("complete WhatsApp intake: %w", err)
	}
	return reply, nil
}

func generateWhatsAppSyntheticPatientReference() (string, error) {
	randomBytes := make([]byte, 9)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return "SYN-WA-" + strings.ToUpper(base64.RawURLEncoding.EncodeToString(randomBytes)), nil
}

func buildWhatsAppUploadURL(frontendURL, token string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(frontendURL))
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid FRONTEND_PUBLIC_URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/upload-user"
	parsed.RawPath = ""
	parsed.Fragment = token
	return parsed.String(), nil
}

func configuredWhatsAppWebhookSecret() string {
	parsed, err := url.Parse(strings.TrimSpace(os.Getenv("FONNTE_WEBHOOK_URL")))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Query().Get("secret"))
}

func validWebhookURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

func generatePublicUploadToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}
