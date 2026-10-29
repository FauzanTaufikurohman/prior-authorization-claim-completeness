package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const fonnteSendEndpoint = "https://api.fonnte.com/send"

func NormalizeWhatsAppNumber(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value[1:], "+") {
		return "", errors.New("nomor WhatsApp tidak valid")
	}

	var digits strings.Builder
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
			digits.WriteRune(character)
		case character == '+' || character == ' ' || character == '-' ||
			character == '(' || character == ')':
		default:
			return "", errors.New("nomor WhatsApp hanya boleh berisi angka dan tanda baca umum")
		}
	}

	number := digits.String()
	if strings.HasPrefix(number, "0") {
		number = "62" + number[1:]
	}
	if !strings.HasPrefix(number, "628") || len(number) < 10 || len(number) > 15 {
		return "", errors.New("gunakan nomor WhatsApp Indonesia yang diawali 08 atau +62")
	}

	return number, nil
}

func BuildPAStatusMessage(requestNumber, status, action, decision string) string {
	var update string
	if action == "record_decision" || action == "record_appeal_decision" {
		decisionLabels := map[string]string{
			"approved":           "disetujui",
			"denied":             "ditolak",
			"partially_approved": "disetujui sebagian",
			"withdrawn":          "ditarik",
		}
		label, ok := decisionLabels[decision]
		if !ok {
			label = "diperbarui"
		}
		if action == "record_appeal_decision" {
			update = "Hasil banding " + label + "."
		} else {
			update = "Keputusan payer: " + label + "."
		}
	} else {
		statusLabels := map[string]string{
			"draft":                   "masih dalam tahap persiapan",
			"ready_for_review":        "menunggu pemeriksaan",
			"in_review":               "sedang diperiksa",
			"needs_information":       "memerlukan informasi tambahan",
			"ready_for_submission":    "siap diajukan ke payer",
			"submitted":               "sudah diajukan ke payer",
			"payer_needs_information": "payer meminta informasi tambahan",
			"completed":               "selesai",
			"cancelled":               "dibatalkan",
			"appeal_submitted":        "banding sudah diajukan",
		}
		label, ok := statusLabels[status]
		if !ok {
			label = "diperbarui"
		}
		update = "Status permohonan " + label + "."
	}

	if strings.TrimSpace(requestNumber) == "" {
		return update + " Hubungi petugas yang menangani permohonan jika ada pertanyaan."
	}
	return "Pembaruan PA " + requestNumber + ": " + update + " Hubungi petugas yang menangani permohonan jika ada pertanyaan."
}

func SendFonnteMessage(ctx context.Context, token, target, message string) error {
	return sendFonnteMessage(ctx, fonnteSendEndpoint, token, target, message)
}

func sendFonnteMessage(ctx context.Context, endpoint, token, target, message string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("FONNTE_ACCOUNT_TOKEN belum dikonfigurasi")
	}

	form := url.Values{}
	form.Set("target", target)
	form.Set("message", message)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("create Fonnte request: %w", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send Fonnte message: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read Fonnte response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Fonnte returned HTTP %d", response.StatusCode)
	}

	var result struct {
		Status bool `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode Fonnte response: %w", err)
	}
	if !result.Status {
		return errors.New("Fonnte did not confirm message delivery")
	}

	return nil
}
