package enclaveproto

type SessionInitRequest struct {
	// Client's ephemeral public key
	ClientEphemeralPublicKey []byte `json:"client_ephemeral_public_key"`
	// Session nonce
	Nonce []byte `json:"nonce"`
}

type SessionInitResponse struct {
	// Session ID
	SessionId string `json:"session_id"`
	// Enclave's ephemeral public key
	EnclaveEphemeralPublicKey []byte `json:"enclave_ephemeral_public_key"`
	// Attestation document
	Attestation []byte `json:"attestation"`
	// Session expiration time, RFC3339 format
	ExpiresAt string `json:"expires_at"`
}
