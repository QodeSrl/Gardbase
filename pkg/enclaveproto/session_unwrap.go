package enclaveproto

type SessionUnwrapItem struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext"`
}

type SessionUnwrapRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id,omitempty"`
	// KMS Key ID
	KeyId string              `json:"key_id,omitempty"`
	Items []SessionUnwrapItem `json:"items,omitempty"`
}

type SessionUnwrapItemResult struct {
	// Object ID
	ObjectId string `json:"object_id,omitempty"`
	// Decrypted DEK, Base64-encoded
	SealedDEK string `json:"sealed_dek,omitempty"`
	// Nonce used for decryption, Base64-encoded
	Nonce   string `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SessionUnwrapResponse []SessionUnwrapItemResult
