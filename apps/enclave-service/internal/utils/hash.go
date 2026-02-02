package utils

import (
	"crypto/sha256"
	"encoding/base64"
)

func Hash(data []byte, salt []byte) string {
	h := sha256.New()
	h.Write(data)
	h.Write(salt)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
