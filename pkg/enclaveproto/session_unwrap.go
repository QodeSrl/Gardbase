package enclaveproto

type SessionUnwrapItem struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Encrypted DEK
	Ciphertext []byte `json:"ciphertext"`
}

type SessionUnwrapRequest struct {
	// Session ID
	SessionId string              `json:"session_id"`
	Items     []SessionUnwrapItem `json:"items"`
}

type SessionUnwrapItemResult struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Decrypted DEK
	SealedDEK []byte `json:"sealed_dek"`
	// Nonce used for decryption
	Nonce   []byte `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SessionUnwrapResponse []SessionUnwrapItemResult
