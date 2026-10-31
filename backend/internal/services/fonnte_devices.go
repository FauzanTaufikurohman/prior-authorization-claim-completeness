package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	fonnteAddDeviceEndpoint     = "https://api.fonnte.com/add-device"
	fonnteUpdateDeviceEndpoint  = "https://api.fonnte.com/update-device"
	fonnteQRCodeEndpoint        = "https://api.fonnte.com/qr"
	fonnteDeviceProfileEndpoint = "https://api.fonnte.com/device"
	fonnteGetDevicesEndpoint    = "https://api.fonnte.com/get-devices"
)

type FonnteDeviceProfile struct {
	Device       string `json:"device"`
	DeviceStatus string `json:"device_status"`
	Name         string `json:"name"`
}

func FindFonnteDeviceToken(ctx context.Context, accountToken, device string) (string, bool, error) {
	return findFonnteDeviceToken(ctx, fonnteGetDevicesEndpoint, accountToken, device)
}

func findFonnteDeviceToken(
	ctx context.Context,
	endpoint, accountToken, device string,
) (string, bool, error) {
	body, err := postFonnteDeviceAPI(ctx, endpoint, accountToken, nil)
	if err != nil {
		return "", false, fmt.Errorf("list Fonnte devices: %w", err)
	}

	var result struct {
		Status bool   `json:"status"`
		Reason string `json:"reason"`
		Data   []struct {
			Device string `json:"device"`
			Token  string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, fmt.Errorf("decode Fonnte devices response: %w", err)
	}
	if !result.Status {
		return "", false, fonnteResponseError("list devices", result.Reason)
	}
	for _, current := range result.Data {
		if current.Device == device {
			if strings.TrimSpace(current.Token) == "" {
				return "", false, errors.New("Fonnte device token is missing")
			}
			return strings.TrimSpace(current.Token), true, nil
		}
	}
	return "", false, nil
}

func AddFonnteDevice(ctx context.Context, accountToken, name, device string) (string, error) {
	return addFonnteDevice(ctx, fonnteAddDeviceEndpoint, accountToken, name, device)
}

func addFonnteDevice(ctx context.Context, endpoint, accountToken, name, device string) (string, error) {
	body, err := postFonnteDeviceAPI(ctx, endpoint, accountToken, map[string]string{
		"name":     name,
		"device":   device,
		"autoread": "true",
		"personal": "true",
		"group":    "false",
	})
	if err != nil {
		return "", fmt.Errorf("add Fonnte device: %w", err)
	}

	var result struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode Fonnte add-device response: %w", err)
	}
	if !result.Status || strings.TrimSpace(result.Token) == "" {
		return "", fonnteResponseError("add device", result.Reason)
	}
	return strings.TrimSpace(result.Token), nil
}

func ConfigureFonnteDevice(ctx context.Context, deviceToken, name, device, webhookURL string) error {
	return configureFonnteDevice(ctx, fonnteUpdateDeviceEndpoint, deviceToken, name, device, webhookURL)
}

func configureFonnteDevice(
	ctx context.Context,
	endpoint, deviceToken, name, device, webhookURL string,
) error {
	body, err := postFonnteDeviceAPI(ctx, endpoint, deviceToken, map[string]string{
		"name":     name,
		"device":   device,
		"webhook":  webhookURL,
		"autoread": "true",
		"personal": "true",
		"group":    "false",
	})
	if err != nil {
		return fmt.Errorf("configure Fonnte device: %w", err)
	}
	var result struct {
		Status bool   `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode Fonnte update-device response: %w", err)
	}
	if !result.Status {
		return fonnteResponseError("configure device", result.Reason)
	}
	return nil
}

func GetFonnteDeviceQRCode(ctx context.Context, deviceToken string) (string, error) {
	return getFonnteDeviceQRCode(ctx, fonnteQRCodeEndpoint, deviceToken)
}

func getFonnteDeviceQRCode(ctx context.Context, endpoint, deviceToken string) (string, error) {
	body, err := postFonnteDeviceAPI(ctx, endpoint, deviceToken, map[string]string{"type": "qr"})
	if err != nil {
		return "", fmt.Errorf("get Fonnte device QR code: %w", err)
	}

	var result struct {
		Status bool   `json:"status"`
		URL    string `json:"url"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode Fonnte QR response: %w", err)
	}
	if !result.Status || strings.TrimSpace(result.URL) == "" {
		return "", fonnteResponseError("get QR code", result.Reason)
	}
	return strings.TrimSpace(result.URL), nil
}

func GetFonnteDeviceProfile(ctx context.Context, deviceToken string) (FonnteDeviceProfile, error) {
	return getFonnteDeviceProfile(ctx, fonnteDeviceProfileEndpoint, deviceToken)
}

func getFonnteDeviceProfile(ctx context.Context, endpoint, deviceToken string) (FonnteDeviceProfile, error) {
	body, err := postFonnteDeviceAPI(ctx, endpoint, deviceToken, nil)
	if err != nil {
		return FonnteDeviceProfile{}, fmt.Errorf("get Fonnte device profile: %w", err)
	}

	var result struct {
		Status bool   `json:"status"`
		Reason string `json:"reason"`
		Device string `json:"device"`
		State  string `json:"device_status"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return FonnteDeviceProfile{}, fmt.Errorf("decode Fonnte device profile response: %w", err)
	}
	if !result.Status || strings.TrimSpace(result.State) == "" {
		return FonnteDeviceProfile{}, fonnteResponseError("get device profile", result.Reason)
	}
	return FonnteDeviceProfile{Device: result.Device, DeviceStatus: result.State, Name: result.Name}, nil
}

func postFonnteDeviceAPI(
	ctx context.Context,
	endpoint, token string,
	fields map[string]string,
) ([]byte, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("Fonnte token is not configured")
	}

	var body io.Reader
	contentType := ""
	if len(fields) > 0 {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				return nil, fmt.Errorf("write Fonnte request field: %w", err)
			}
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("finish Fonnte request: %w", err)
		}
		body = &buffer
		contentType = writer.FormDataContentType()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create Fonnte request: %w", err)
	}
	request.Header.Set("Authorization", token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send Fonnte request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Fonnte response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Fonnte returned HTTP %d", response.StatusCode)
	}
	return responseBody, nil
}

func fonnteResponseError(operation, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("Fonnte could not %s", operation)
	}
	return fmt.Errorf("Fonnte could not %s: %s", operation, reason)
}
