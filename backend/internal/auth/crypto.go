package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

// GenerateRandomBytes returns n cryptographically secure random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.New("failed to read secure random bytes: " + err.Error())
	}
	return b, nil
}

// GenerateRandomHex returns a hex-encoded string of n random bytes
func GenerateRandomHex(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateTVSecret produces a 256-bit secret for Fire TV authentication
func GenerateTVSecret() (string, error) {
	return GenerateRandomHex(32) // 256 bits
}

// GenerateJoinToken produces a 128-bit random token for QR join
func GenerateJoinToken() (string, error) {
	return GenerateRandomHex(16) // 128 bits
}

// GenerateSessionToken produces a 256-bit token for phone browser sessions
func GenerateSessionToken() (string, error) {
	return GenerateRandomHex(32) // 256 bits
}

// HashSecret computes a SHA-256 hex digest of a raw secret
func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CompareHash performs constant-time comparison of two hashes
func CompareHash(hashA, hashB string) bool {
	if len(hashA) != len(hashB) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hashA), []byte(hashB)) == 1
}
