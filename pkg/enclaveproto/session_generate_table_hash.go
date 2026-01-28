package enclaveproto

type EnclaveSessionGenerateTableHashRequest struct {
	// Session ID, Base64-encoded
	SessionID string `json:"session_id"`
	// Session encrypted table name to decrypt and re-encrypt Base64-encoded
	SessionEncryptedTableName string `json:"session_encrypted_table_name"`
	// Nonce for session encrypted table name, Base64-encoded
	SessionTableNameNonce string `json:"session_table_name_nonce"`
	// Wrapped Table Salt
	TableSalt string `json:"table_salt"`
}

type EnclaveSessionGenerateTableHashResponse struct {
	// New table hash
	TableHash string `json:"table_hash"`
}
