package core

import (
	"crypto/rand"
	"fmt"
)

// GenerateID creates a random unique ID with a prefix.
// Example: nat-3f9c2a1b8e
func GenerateID(prefix string) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		return prefix + "-xxxxxx"
	}
	return fmt.Sprintf("%s-%x", prefix, b)
}
