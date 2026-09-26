package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
