package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

const (
	AESKeySize   = 32 // 256-bit AES key size
	GMCNonceSize = 12 // GCM standard nonce size
)

func generateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func deriveNonceHMAC(key []byte, context string, contextType string, nonceSize int) []byte {
	// derive nonce using HMAC-SHA256
	h := hmac.New(sha256.New, key)

	h.Write([]byte(contextType))
	h.Write([]byte{0x00}) // separator
	h.Write([]byte(context))

	hash := h.Sum(nil)

	nonce := make([]byte, nonceSize)
	copy(nonce, hash[:nonceSize])

	return nonce
}