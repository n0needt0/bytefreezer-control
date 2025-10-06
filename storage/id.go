package storage

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

// GenerateShortID generates a short, URL-safe ID (12 characters)
// Format: lowercase base32 without padding
// Example: "a3k7m2pqw5xz"
func GenerateShortID() string {
	// Generate 8 random bytes (64 bits of entropy)
	// This gives us ~1.8 quintillion possible IDs
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // Should never happen
	}

	// Encode to base32 and take first 12 chars
	encoded := base32.StdEncoding.EncodeToString(b)
	// Remove padding and lowercase
	id := strings.ToLower(strings.TrimRight(encoded, "="))

	// Take first 12 characters
	if len(id) > 12 {
		id = id[:12]
	}

	return id
}
