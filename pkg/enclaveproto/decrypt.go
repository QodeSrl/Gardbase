package enclaveproto

type DecryptRequest struct {
	// Encrypted DEK
	Ciphertext []byte `json:"ciphertext"`
	// Client's ephemeral public key
	ClientEphemeralPublicKey []byte `json:"client_ephemeral_public_key"`
	// Request nonce
	Nonce []byte `json:"nonce"`
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
}
