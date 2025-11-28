package enclaveproto

type SessionGenerateDEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id,omitempty"`
	// KMS Key ID
	KeyId string `json:"key_id,omitempty"`
	// Number of DEKs to generate
	Count int `json:"count,omitempty"`
}

type GeneratedDEK struct {
	// Generated DEK, Base64-encoded
	SealedDEK string `json:"dek,omitempty"`
	// Generated DEK encrypted by KMS, Base64-encoded
	KmsEncryptedDEK string `json:"kms_encrypted_dek,omitempty"`
	// Nonce used for encryption, Base64-encoded
	Nonce string `json:"nonce,omitempty"`
}

type SessionGenerateDEKResponse struct {
	// List of generated DEKs
	DEKs []GeneratedDEK `json:"deks,omitempty"`
}
