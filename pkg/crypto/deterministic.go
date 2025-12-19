// Deterministic encryption using AES-GMC with HMAC-derived nonces
// ct format: gcmct

// IMPORTANT: This provides deterministic encryption where the same plaintext with
// the same context always produces the same ciphertext. The nonce is derived using
// HMAC-SHA256 from the key and context, ensuring determinism while maintaining
// cryptographic properties. Different contexts will yield different ciphertexts for the same plaintext.

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

var contextTypeDeterministicEncryption = "gardbase-data-deterministic-encryption-v1"

func EncryptObjectDeterministic(pt []byte, context string, dek []byte) ([]byte, error) {
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid DEK size: %d", len(dek))
	}
	if context == "" {
		return nil, fmt.Errorf("context must not be empty for deterministic encryption")
	}

	// encrypt pt with DEK using context-derived nonce
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := deriveNonceHMAC(dek, context, contextTypeDeterministicEncryption, gcm.NonceSize())
	ct := gcm.Seal(nil, nonce, pt, []byte(context))

	return ct, nil
}

func DecryptObjectDeterministic(ct []byte, context string, dek []byte) ([]byte, error) {
	if context == "" {
		return nil, fmt.Errorf("context must not be empty for deterministic decryption")
	}

	// decrypt ct with DEK using context-derived nonce
	if len(dek) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(dek))
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := deriveNonceHMAC(dek, context, contextTypeDeterministicEncryption, gcm.NonceSize())
	pt, err := gcm.Open(nil, nonce, ct, []byte(context))
	if err != nil {
		return nil, err
	}
	return pt, nil
}
