package services

import (
	"regexp"
	"testing"
	"time"
)

func TestGenerateRequestNumber(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)

	first, err := GenerateRequestNumber(now)
	if err != nil {
		t.Fatalf("GenerateRequestNumber() error = %v", err)
	}

	if !regexp.MustCompile(`^PA-20261025-[a-f0-9]{8}$`).MatchString(first) {
		t.Fatalf("unexpected request number format: %s", first)
	}

	second, err := GenerateRequestNumber(now)
	if err != nil {
		t.Fatalf("GenerateRequestNumber() error = %v", err)
	}

	if first == second {
		t.Fatal("expected independently generated request numbers to differ")
	}
}
