package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	password := "Correct-Horse-Battery-42"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == password {
		t.Fatal("password must not be stored in plaintext")
	}

	if !VerifyPassword(hash, password) {
		t.Fatal("VerifyPassword() rejected the correct password")
	}

	if VerifyPassword(hash, "wrong-password-value") {
		t.Fatal("VerifyPassword() accepted an incorrect password")
	}
}

func TestHashPasswordRejectsShortPassword(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected an error for a short password")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if VerifyPassword("not-a-valid-hash", "some-password") {
		t.Fatal("malformed hash should not verify")
	}
}
