package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func GenerateRequestNumber(now time.Time) (string, error) {
	randomBytes := make([]byte, 4)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate request number: %w", err)
	}

	return fmt.Sprintf(
		"PA-%s-%s",
		now.UTC().Format("20060102"),
		hex.EncodeToString(randomBytes),
	), nil
}
