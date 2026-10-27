package auth

import "crypto/rand"

func secureRandom(destination []byte) (int, error) {
	return rand.Read(destination)
}
