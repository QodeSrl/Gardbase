package enclaveproto

type DecryptRequest struct {
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext,omitempty"`
	// Client's ephemeral public key
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key,omitempty"`
	// Request nonce, Base64-encoded
	Nonce string `json:"nonce"`
	// KMS Key ID
	KeyID string `json:"key_id,omitempty"`
}

type DecryptResponse struct {
	// x25519 public key of the enclave, Base64-encoded
	EnclavePubKey string `json:"enclave_public_key,omitempty"`
	// Unwrapped DEK, encrypted with NaCl box, Base64-encoded
	Ciphertext string `json:"ciphertext,omitempty"`
	// Nonce used for NaCl box encryption, Base64-encoded
	Nonce string `json:"nonce,omitempty"`
	// Request nonce, Base64-encoded
	RequestNonce string `json:"request_nonce"`
	// Attestation used for KMS decryption, Base64-encoded
	Attestation string `json:"attestation,omitempty"`
}
