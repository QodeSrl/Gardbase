package enclaveproto

type DecryptRequest struct {
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext"`
	// Client's ephemeral public key
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key"`
	// Request nonce, Base64-encoded
	Nonce string `json:"nonce"`
	// KMS Key ID
	KeyID string `json:"key_id"`
}

type DecryptResponse struct {
	// x25519 public key of the enclave, Base64-encoded
	EnclavePubKey string `json:"enclave_public_key"`
	// Unwrapped DEK, encrypted with NaCl box, Base64-encoded
	Ciphertext string `json:"ciphertext"`
	// Nonce used for NaCl box encryption, Base64-encoded
	Nonce string `json:"nonce"`
	// Request nonce, Base64-encoded
	RequestNonce string `json:"request_nonce"`
	// Attestation used for KMS decryption, Base64-encoded
	Attestation string `json:"attestation"`
}
