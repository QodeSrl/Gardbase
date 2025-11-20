// Probabilistic encryption using AES-GCM.
// ct format: nonce(12) || gcmct

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

func EncryptObjectProbabilistic(dek []byte, pt []byte) (cipherText []byte, err error) {
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid DEK size: %d", len(dek))
	}

	// encrypt pt with DEK
	block, err := aes.NewCipher(dek)
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

	return ct, nil
}

// TODO: implement KMS and Nitro Enclaves integration
func DecryptObjectProbabilistic(masterKey, ct, encryptedDEK []byte) (plainText []byte, err error) {
	if len(masterKey) != AESKeySize {
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
