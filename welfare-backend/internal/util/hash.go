package util

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256String(input string) string {
	s := sha256.Sum256([]byte(input))
	return hex.EncodeToString(s[:])
}
