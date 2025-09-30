// Probabilistic encryption using AES-GCM.
// Master key: 32 bytes symmetric key used to encrypt tenant's DEKs.
// DEK: Data Encryption Key, randomly generate 32 bytes symmetric key used to encrypt data.
// ct format: nonce(12) || gcmct
// encrypted DEK format: nonce(12) || gcmct

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
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

// TODO: implement KMS integration
func EncryptObjectProbabilistic(masterKey, pt []byte) (cipherText []byte, encryptedDEK []byte, err error) {
	if (len(masterKey) != AESKeySize) {
		return nil, nil, fmt.Errorf("invalid master key size: %d", len(masterKey))
	}

	// generate a random DEK
	dek, err := generateRandomBytes(AESKeySize)
	if err != nil {
		return nil, nil, err
	}

	// encrypt pt with DEK
	ct, err := aesGMCEncrypt(dek, pt)
	if err != nil {
		return nil, nil, err
	}

	// encrypt DEK with master key
	encryptedDEK, err = aesGMCEncrypt(masterKey, dek)
	if err != nil {
		return nil, nil, err
	}

	return ct, encryptedDEK, nil
}

func DecryptObjectProbabilistic(masterKey, ct, encryptedDEK []byte) (plainText []byte, err error) {
	if (len(masterKey) != AESKeySize) {
		return nil, fmt.Errorf("invalid master key size: %d", len(masterKey))
	}
	// decrypt DEK with master key
	dek, err := aesGCMDecrypt(masterKey, encryptedDEK)
	if err != nil {
		return nil, err
	}
	// decrypt ct with DEK
	pt, err := aesGCMDecrypt(dek, ct)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func aesGMCEncrypt(key, pt []byte) ([]byte, error) {
	if len(key) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gmc, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := generateRandomBytes(GMCNonceSize)
	if err != nil {
		return nil, err
	}
	ct := gmc.Seal(nil, nonce, pt, nil)
	// return nonce || ct
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func aesGCMDecrypt(key, input []byte) ([]byte, error) {
	if len(input) < GMCNonceSize {
		return nil, errors.New("input too short")
	}
	if len(key) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}
	nonce := input[:GMCNonceSize]
	ct := input[GMCNonceSize:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gmc, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt, err := gmc.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}