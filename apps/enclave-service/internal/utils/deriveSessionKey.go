package utils

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

func DeriveSessionKey(ephPriv [32]byte, clientPub []byte) ([]byte, error) {
	if len(clientPub) != 32 {
		return nil, fmt.Errorf("invalid client public key length")
	}
	shared, err := curve25519.X25519(ephPriv[:], clientPub)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %v", err)
	}
	hk := hkdf.New(sha256.New, shared, nil, []byte("gardbase-enclave-session-v1"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(hk, key); err != nil {
		return nil, fmt.Errorf("failed to derive session key: %v", err)
	}
	Zero(shared)
	return key, nil
}
