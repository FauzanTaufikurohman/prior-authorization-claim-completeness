package handlers

import (
	"encoding/base64"
	"testing"
)

func TestParsePublicUploadToken(t *testing.T) {
	rawToken := make([]byte, 32)
	token := base64.RawURLEncoding.EncodeToString(rawToken)

	hash, ok := parsePublicUploadToken(token)
	if !ok {
		t.Fatal("expected a 32-byte URL-safe token to be accepted")
	}
	if len(hash) != 64 {
		t.Fatalf("expected a 64-character SHA-256 hash, got %d", len(hash))
	}

	for _, invalid := range []string{"", "not-a-token", base64.RawURLEncoding.EncodeToString(rawToken[:16])} {
		if _, ok := parsePublicUploadToken(invalid); ok {
			t.Errorf("expected token %q to be rejected", invalid)
		}
	}
}
