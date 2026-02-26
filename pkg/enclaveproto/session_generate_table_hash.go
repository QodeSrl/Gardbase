package enclaveproto

type SessionGenerateTableHashRequest struct {
	// Session ID
	SessionID string `json:"session_id"`
	// Session encrypted table name to decrypt and re-encrypt
	SessionEncryptedTableName []byte `json:"session_encrypted_table_name"`
	// Nonce for session encrypted table name,
	SessionTableNameNonce []byte `json:"session_table_name_nonce"`
	// Wrapped Table Salt
	TableSalt []byte `json:"table_salt"`
}

type SessionGenerateTableHashResponse struct {
	// New table hash
	TableHash string `json:"table_hash"`
}
