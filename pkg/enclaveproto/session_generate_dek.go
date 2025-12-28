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

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclaveDEKToPrepare struct {
	// KMS Encrypted DEK, Base64-encoded
	CiphertextBlob string `json:"ciphertext_blob"`
	// KMS Ciphertext for Recipient, Base64-encoded
	CiphertextForRecipient string `json:"ciphertext_for_recipient"`
}

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclavePrepareDEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// List of DEKs to prepare
	DEKs []EnclaveDEKToPrepare `json:"deks"`
}

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclavePrepareDEKResponse struct {
	// List of prepared DEKs
	DEKs []GeneratedDEK `json:"deks"`
}
