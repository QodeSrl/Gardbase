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

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclaveSessionUnwrapItem struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext"`
}

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclaveSessionUnwrapRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// KMS Key ID
	KeyId string `json:"key_id"`
	// List of items to unwrap
	Items []EnclaveSessionUnwrapItem `json:"items"`
}

// Note: All structs that start with "Enclave" are used for communication between the API service and the enclave.
type EnclaveSessionUnwrapResponse []SessionUnwrapItemResult
