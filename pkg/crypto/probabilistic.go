// Probabilistic encryption using AES-GCM.
// Master key: 32 bytes symmetric key used to encrypt tenant's DEKs.
// DEK: Data Encryption Key, randomly generate 32 bytes symmetric key used to encrypt data.
// ct format: nonce(12) || gcmct
// encrypted DEK format: nonce(12) || gcmct

package crypto

import (
	"crypto/rand"
)

const (
	AESKeySize = 32 // 256-bit AES key size
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
