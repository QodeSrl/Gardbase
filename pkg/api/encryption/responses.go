package encryption

import "github.com/qodesrl/gardbase/pkg/enclaveproto"

type SessionInitResponse = enclaveproto.SessionInitResponse

type SessionUnwrapResponse = enclaveproto.SessionUnwrapResponse

type SessionGenerateDEKResponse struct {
	// List of generated DEKs
	DEKs []enclaveproto.GeneratedDEK `json:"deks"`
	// Index encryption key (IEK), Base64-encoded
	SealedIEK []byte `json:"iek"`
	// IEK Nonce used for sealing, Base64-encoded
	IEKNonce []byte `json:"iek_nonce"`
}

type DecryptResponse struct {
	// x25519 public key of the enclave
	EnclavePubKey []byte `json:"enclave_public_key"`
	// Unwrapped DEK, encrypted with NaCl box
	Ciphertext []byte `json:"ciphertext"`
	// Nonce used for NaCl box encryption
	Nonce []byte `json:"nonce"`
	// Request nonce
	RequestNonce []byte `json:"request_nonce"`
	// Attestation document from the enclave
	Attestation []byte `json:"attestation"`
}
