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

const (
	contextDEKEncryption = "dek-encryption"
	contextDataEncryption = "data-encryption"
)

func EncryptObjectDeterministic(dek []byte, pt []byte, context string) (cipherText []byte, err error) {
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

	nonce := deriveNonceHMAC(dek, context, contextDataEncryption, gcm.NonceSize())
	ct := gcm.Seal(nil, nonce, pt, []byte(context))

	return ct, nil
}

// TODO: implement KMS and Nitro Enclaves integration
func DecryptObjectDeterministic(masterKey, ct, encryptedDEK []byte, context string) (plainText []byte, err error) {
	if (len(masterKey) != AESKeySize) {
		return nil, fmt.Errorf("invalid master key size: %d", len(masterKey))
	}
	if context == "" {
		return nil, fmt.Errorf("context must not be empty for deterministic decryption")
	}

	// decrypt DEK with master key using context-derived nonce
	dekContext := fmt.Sprintf("%s:%s", context, "dek")
	dek, err := aesGCMDecryptDeterministic(masterKey, encryptedDEK, dekContext, contextDEKEncryption)
	if err != nil {
		return nil, err
	}

	// decrypt ct with DEK using context-derived nonce
	pt, err := aesGCMDecryptDeterministic(dek, ct, context, contextDataEncryption)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

func aesGCMDecryptDeterministic(key, ct []byte, context string, contextType string) ([]byte, error) {
	if len(key) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := deriveNonceHMAC(key, context, contextType, gcm.NonceSize())
	pt, err := gcm.Open(nil, nonce, ct, []byte(context))
	if err != nil {
		return nil, err
	}
	return pt, nil
}
