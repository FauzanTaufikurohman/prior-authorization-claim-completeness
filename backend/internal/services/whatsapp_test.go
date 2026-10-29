package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeWhatsAppNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "local Indonesian number", input: "0812-3456-7890", want: "6281234567890"},
		{name: "international Indonesian number", input: "+62 812 3456 7890", want: "6281234567890"},
		{name: "already normalized", input: "6281234567890", want: "6281234567890"},
		{name: "unsupported country", input: "+1 202 555 0100", wantErr: true},
		{name: "letters", input: "0812abc", wantErr: true},
		{name: "too short", input: "0812", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeWhatsAppNumber(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeWhatsAppNumber(%q) expected an error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeWhatsAppNumber(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("NormalizeWhatsAppNumber(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestBuildPAStatusMessage(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		action   string
		decision string
		want     string
	}{
		{
			name:     "approval",
			status:   "completed",
			action:   "record_decision",
			decision: "approved",
			want:     "Pembaruan PA PA-123: Keputusan payer: disetujui. Hubungi petugas yang menangani permohonan jika ada pertanyaan.",
		},
		{
			name:   "cancellation",
			status: "cancelled",
			action: "cancel",
			want:   "Pembaruan PA PA-123: Status permohonan dibatalkan. Hubungi petugas yang menangani permohonan jika ada pertanyaan.",
		},
		{
			name:     "appeal decision",
			status:   "completed",
			action:   "record_appeal_decision",
			decision: "denied",
			want:     "Pembaruan PA PA-123: Hasil banding ditolak. Hubungi petugas yang menangani permohonan jika ada pertanyaan.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildPAStatusMessage("PA-123", test.status, test.action, test.decision)
			if got != test.want {
				t.Errorf("BuildPAStatusMessage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSendFonnteMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "test-token" {
			t.Errorf("Authorization header = %q, want test-token", request.Header.Get("Authorization"))
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if got := request.Form.Get("target"); got != "6281234567890" {
			t.Errorf("target = %q, want normalized phone", got)
		}
		if got := request.Form.Get("message"); got != "Pembaruan PA PA-123" {
			t.Errorf("message = %q, want status message", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true}`))
	}))
	defer server.Close()

	if err := sendFonnteMessage(
		context.Background(),
		server.URL,
		"test-token",
		"6281234567890",
		"Pembaruan PA PA-123",
	); err != nil {
		t.Fatalf("sendFonnteMessage() error = %v", err)
	}
}
