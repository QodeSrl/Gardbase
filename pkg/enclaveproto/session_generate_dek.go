package enclaveproto

type SessionGenerateDEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// KMS Key ID
	KeyId string `json:"key_id"`
	// Number of DEKs to generate
	Count int `json:"count"`
}

type GeneratedDEK struct {
	// Generated DEK, Base64-encoded
	SealedDEK string `json:"dek"`
	// Generated DEK encrypted by KMS, Base64-encoded
	KmsEncryptedDEK string `json:"kms_encrypted_dek"`
	// Nonce used for encryption, Base64-encoded
	Nonce string `json:"nonce"`
}

type SessionGenerateDEKResponse struct {
	// List of generated DEKs
	DEKs []GeneratedDEK `json:"deks"`
}
