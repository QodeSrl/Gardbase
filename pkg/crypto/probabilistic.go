// Probabilistic encryption using AES-GCM.
// ct format: nonce(12) || gcmct

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

func EncryptObjectProbabilistic(pt []byte, dek []byte) ([]byte, error) {
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

func DecryptObjectProbabilistic(ct []byte, dek []byte) ([]byte, error) {
	if len(ct) < GMCNonceSize {
		return nil, errors.New("input too short")
	}
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(dek))
	}
	nonce := ct[:GMCNonceSize]
	ct = ct[GMCNonceSize:]
	block, err := aes.NewCipher(dek)
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
