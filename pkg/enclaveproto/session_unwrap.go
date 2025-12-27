package enclaveproto

type SessionUnwrapItem struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext"`
}

type SessionUnwrapRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// KMS Key ID
	KeyId string              `json:"key_id"`
	Items []SessionUnwrapItem `json:"items"`
}

type SessionUnwrapItemResult struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Decrypted DEK, Base64-encoded
	SealedDEK string `json:"sealed_dek"`
	// Nonce used for decryption, Base64-encoded
	Nonce   string `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SessionUnwrapResponse []SessionUnwrapItemResult
