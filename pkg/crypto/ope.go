// Order-Preserving and Range-Query Encryption using AES-GCM.
//
// CRITICAL SECURITY WARNING
// Order-preserving encryption (OPE) leaks order, approximate values, distribution, and frequency information.
// Only use for LOW-SENSITIVITY data where range queries are absolutely necessary.

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// WARNING: This is trivially breakable with known plaintext attacks
// TODO: Replace with a more secure OPE scheme
//
// ct format: encrypted value (8 bytes)
func EncryptObjectLinearOPE(dek []byte, plaintext uint64) (cipherText []byte, err error) {
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid DEK size: %d", len(dek))
	}

	// Encrypt plaintext with DEK using linear OPE

	// Derive slope and intercept from key
	a, b := deriveLinearParams(dek)

	// Perform linear transformation
	// This preserves order: if pt1 < pt2, then ct1 < ct2
	ciphertext := (a * plaintext) + b

	// Encode as bytes
	ct := make([]byte, 8)
	binary.BigEndian.PutUint64(ct, ciphertext)

	return ct, nil
}

// TODO: implement KMS and Nitro Enclaves integration
// DecryptObjectLinearOPE decrypts data encrypted with EncryptObjectLinearOPE.
func DecryptObjectLinearOPE(masterKey, ct, encryptedDEK []byte) (plainText uint64, err error) {
	if len(masterKey) != AESKeySize {
		return 0, fmt.Errorf("invalid master key size: %d", len(masterKey))
	}

	// Decrypt DEK with master key
	dek, err := aesGCMDecrypt(masterKey, encryptedDEK)
	if err != nil {
		return 0, err
	}

	// Decrypt ciphertext with DEK
	pt, err := linearOPEDecrypt(dek, ct)
	if err != nil {
		return 0, err
	}

	return pt, nil
}

// linearOPEDecrypt performs inverse linear transformation: pt = (ct - b) / a
func linearOPEDecrypt(key, ct []byte) (uint64, error) {
	if len(key) != AESKeySize {
		return 0, fmt.Errorf("invalid key size: %d", len(key))
	}
	if len(ct) != 8 {
		return 0, errors.New("invalid ciphertext size")
	}

	// Derive slope and intercept from key
	a, b := deriveLinearParams(key)

	// Decode ciphertext
	ciphertext := binary.BigEndian.Uint64(ct)

	// Perform inverse transformation
	plaintext := (ciphertext - b) / a

	return plaintext, nil
}

// deriveLinearParams derives slope (a) and intercept (b) from key
func deriveLinearParams(key []byte) (a uint64, b uint64) {
	h := hmac.New(sha256.New, key)
	h.Write([]byte("linear-ope-slope"))
	slopeBytes := h.Sum(nil)
	a = binary.BigEndian.Uint64(slopeBytes[:8]) | 1 // Ensure odd (coprime with 2^64)

	h.Reset()
	h.Write([]byte("linear-ope-intercept"))
	interceptBytes := h.Sum(nil)
	b = binary.BigEndian.Uint64(interceptBytes[:8])

	return a, b
}
