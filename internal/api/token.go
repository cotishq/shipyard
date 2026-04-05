package api

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)


func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}