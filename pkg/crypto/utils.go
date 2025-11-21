package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
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

func GenerateEphemeralKeypair() (clientPriv [32]byte, clientPub [32]byte, clientPubB64 string, err error) {
	if _, err := rand.Read(clientPriv[:]); err != nil {
		return clientPriv, clientPub, "", err
	}
	curve25519.ScalarBaseMult(&clientPub, &clientPriv)
	clientPubB64 = base64.StdEncoding.EncodeToString(clientPub[:])
	return clientPriv, clientPub, clientPubB64, nil
}

func deriveSessionKey(clientPriv [32]byte, enclavePubRaw []byte) ([]byte, error) {
	if len(enclavePubRaw) != 32 {
		return nil, errors.New("invalid client public key length")
	}
	shared, err := curve25519.X25519(clientPriv[:], enclavePubRaw)
	if err != nil {
		return nil, err
	}
	hk := hkdf.New(sha256.New, shared, nil, []byte("gardbase-enclave-session-v1"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(hk, key); err != nil {
		return nil, err
	}
	for i := range shared {
		shared[i] = 0
	}
	return key, nil
}

func verifyAttestation(ds *DecryptSession) bool {
	// TODO: implement attestation verification
	fmt.Printf("[WARNING] Attestation verification not implemented yet\n")
	ds.AttestationVerified = true
	return true
}
