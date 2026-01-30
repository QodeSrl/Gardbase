package enclaveproto

type GeneratedDEK struct {
	// Generated DEK, Base64-encoded
	SealedDEK string `json:"dek"`
	// Generated DEK encrypted by KMS, Base64-encoded
	KmsEncryptedDEK string `json:"kms_encrypted_dek"`
	// Generated DEK encrypted by Master Key, Base64-encoded
	MasterEncryptedDEK string `json:"master_encrypted_dek"`
	// Session Nonce used for sealing, Base64-encoded
	SessionNonce string `json:"session_nonce"`
	// Master Key Nonce used for encryption, Base64-encoded
	MasterKeyNonce string `json:"master_key_nonce"`
}

type DEKToPrepare struct {
	// KMS Encrypted DEK, Base64-encoded
	CiphertextBlob string `json:"ciphertext_blob"`
	// KMS Ciphertext for Recipient, Base64-encoded
	CiphertextForRecipient string `json:"ciphertext_for_recipient"`
}

type PrepareDEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// KMS wrapped Master Key, Base64-encoded
	WrappedMasterKey string `json:"wrapped_master_key"`
	// List of DEKs to prepare
	DEKs []DEKToPrepare `json:"deks"`
}

type PrepareDEKResponse struct {
	// List of prepared DEKs
	DEKs []GeneratedDEK `json:"deks"`
}
