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

func TestSendFonnteReplyUsesIncomingInboxID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "device-token" {
			t.Errorf("Authorization header = %q, want device-token", request.Header.Get("Authorization"))
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		if got := request.Form.Get("target"); got != "6281234567890" {
			t.Errorf("target = %q, want sender number", got)
		}
		if got := request.Form.Get("inboxid"); got != "inbox-42" {
			t.Errorf("inboxid = %q, want incoming inbox ID", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":true}`))
	}))
	defer server.Close()

	if err := sendFonnteReply(context.Background(), server.URL, "device-token", "6281234567890", "inbox-42", "Balasan"); err != nil {
		t.Fatalf("sendFonnteReply() error = %v", err)
	}
}
