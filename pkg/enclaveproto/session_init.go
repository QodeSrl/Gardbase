package enclaveproto

type SessionInitRequest struct {
	// Client's ephemeral public key
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key"`
	// Session nonce, Base64-encoded
	Nonce string `json:"nonce"`
}

type SessionInitResponse struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// Enclave's ephemeral public key, Base64-encoded
	EnclaveEphemeralPublicKey string `json:"enclave_ephemeral_public_key"`
	// Attestation document, Base64-encoded
	Attestation string `json:"attestation"`
	// Session expiration time, RFC3339 format
	ExpiresAt string `json:"expires_at"`
}
