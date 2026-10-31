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

func SendFonnteReply(ctx context.Context, token, target, inboxID, message string) error {
	return sendFonnteReply(ctx, fonnteSendEndpoint, token, target, inboxID, message)
}

func sendFonnteReply(ctx context.Context, endpoint, token, target, inboxID, message string) error {
	return sendFonnteMessageWithInbox(ctx, endpoint, token, target, inboxID, message)
}

func sendFonnteMessageWithInbox(
	ctx context.Context,
	endpoint, token, target, inboxID, message string,
) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("Fonnte device token is required")
	}
	if strings.TrimSpace(target) == "" || strings.TrimSpace(message) == "" {
		return errors.New("target and message are required for Fonnte")
	}

	form := url.Values{}
	form.Set("target", target)
	form.Set("message", message)
	if strings.TrimSpace(inboxID) != "" {
		form.Set("inboxid", strings.TrimSpace(inboxID))
	}
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
