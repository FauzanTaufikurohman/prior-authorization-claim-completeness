package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddFonnteDeviceUsesAccountTokenAndReturnsDeviceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "account-token" {
			t.Errorf("Authorization = %q, want account-token", got)
		}
		if !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Errorf("Content-Type = %q, want multipart form data", request.Header.Get("Content-Type"))
		}
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm() error = %v", err)
		}
		for key, want := range map[string]string{
			"name": "PA Staff 4", "device": "6281234567890", "autoread": "true", "personal": "true", "group": "false",
		} {
			if got := request.FormValue(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true,"token":"device-token"}`))
	}))
	defer server.Close()

	token, err := addFonnteDevice(context.Background(), server.URL, "account-token", "PA Staff 4", "6281234567890")
	if err != nil || token != "device-token" {
		t.Fatalf("addFonnteDevice() = (%q, %v), want device-token and no error", token, err)
	}
}

func TestFindFonnteDeviceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "account-token" {
			t.Errorf("Authorization = %q, want account-token", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true,"data":[{"device":"6281234567890","token":"matching-token"}]}`))
	}))
	defer server.Close()

	token, found, err := findFonnteDeviceToken(context.Background(), server.URL, "account-token", "6281234567890")
	if err != nil || !found || token != "matching-token" {
		t.Fatalf("findFonnteDeviceToken() = (%q, %t, %v), want matching device", token, found, err)
	}
}

func TestFonnteDeviceAPIRejectsFailedUpdateAndReturnsQR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/update" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"status":false,"reason":"webhook url invalid"}`))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true,"url":"aGVsbG8="}`))
	}))
	defer server.Close()

	err := configureFonnteDevice(context.Background(), server.URL+"/update", "device-token", "PA Staff 4", "6281234567890", "https://app.example/api/webhooks/whatsapp")
	if err == nil || !strings.Contains(err.Error(), "webhook url invalid") {
		t.Fatalf("configureFonnteDevice() error = %v, want Fonnte rejection reason", err)
	}
	qr, err := getFonnteDeviceQRCode(context.Background(), server.URL+"/qr", "device-token")
	if err != nil || qr != "aGVsbG8=" {
		t.Fatalf("getFonnteDeviceQRCode() = (%q, %v), want base64 payload", qr, err)
	}
}

func TestGetFonnteDeviceProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "device-token" {
			t.Errorf("Authorization = %q, want device-token", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true,"device":"6281234567890","device_status":"connect","name":"PA Staff 4"}`))
	}))
	defer server.Close()

	profile, err := getFonnteDeviceProfile(context.Background(), server.URL, "device-token")
	if err != nil || profile.DeviceStatus != "connect" || profile.Device != "6281234567890" {
		t.Fatalf("getFonnteDeviceProfile() = (%+v, %v), want connected device", profile, err)
	}
}

func TestFonnteTokenEncryptionRoundTripAndTamperDetection(t *testing.T) {
	secret := strings.Repeat("x", 32)
	encrypted, err := EncryptFonnteToken("device-token", secret)
	if err != nil {
		t.Fatalf("EncryptFonnteToken() error = %v", err)
	}
	decrypted, err := DecryptFonnteToken(encrypted, secret)
	if err != nil || decrypted != "device-token" {
		t.Fatalf("DecryptFonnteToken() = (%q, %v), want device-token", decrypted, err)
	}
	if _, err := DecryptFonnteToken(encrypted, strings.Repeat("y", 32)); err == nil {
		t.Fatal("DecryptFonnteToken() accepted a different key")
	}
}
