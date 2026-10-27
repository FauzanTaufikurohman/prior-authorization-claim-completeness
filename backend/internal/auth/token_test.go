package auth

import "testing"

func TestGenerateSessionToken(t *testing.T) {
	token1, hash1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() error = %v", err)
	}

	token2, hash2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() error = %v", err)
	}

	if token1 == "" || hash1 == "" {
		t.Fatal("token and hash must not be empty")
	}

	if token1 == token2 || hash1 == hash2 {
		t.Fatal("separate sessions should have different tokens")
	}

	if HashSessionToken(token1) != hash1 {
		t.Fatal("stored token hash does not match token")
	}
}
