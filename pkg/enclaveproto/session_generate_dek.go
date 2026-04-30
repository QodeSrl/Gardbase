package enclaveproto

type GeneratedDEK struct {
	// Generated DEK
	SealedDEK []byte `json:"dek"`
	// Generated DEK encrypted by KMS
	KmsEncryptedDEK []byte `json:"kms_encrypted_dek"`
	// Generated DEK encrypted by Master Key
	MasterEncryptedDEK []byte `json:"master_encrypted_dek"`
	// Session Nonce used for sealing
	SessionNonce []byte `json:"session_nonce"`
	// Master Key Nonce used for encryption
	MasterKeyNonce []byte `json:"master_key_nonce"`
}

type DEKToPrepare struct {
	// KMS Encrypted DEK
	CiphertextBlob []byte `json:"ciphertext_blob"`
	// KMS Ciphertext for Recipient
	CiphertextForRecipient []byte `json:"ciphertext_for_recipient"`
}

type PrepareDEKRequest struct {
	// Session ID
	SessionId string `json:"session_id"`
	// KMS wrapped Master Key
	WrappedMasterKey []byte `json:"wrapped_master_key"`
	// List of DEKs to prepare
	DEKs []DEKToPrepare `json:"deks"`
	// IEK to prepare
	IEK []byte `json:"iek"`
}

type PrepareDEKResponse struct {
	// List of prepared DEKs
	DEKs []GeneratedDEK `json:"deks"`
	// Index encryption key (IEK)
	SealedIEK []byte `json:"iek"`
	// IEK Nonce used for sealing
	IEKNonce []byte `json:"iek_nonce"`
}
