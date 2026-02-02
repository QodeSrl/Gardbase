package objects

type GetTableHashRequest struct {
	SessionID                 string `json:"session_id" binding:"required"` // Base64-encoded session ID
	SessionEncryptedTableName string `json:"encrypted_table_name,omitempty"`
	SessionTableNameNonce     string `json:"table_name_nonce,omitempty"`
}

type CreateObjectRequest struct {
	BlobSize           int64             `json:"blob_size" binding:"required"`
	KMSEncryptedDEK    string            `json:"encrypted_dek" binding:"required"`
	MasterEncryptedDEK string            `json:"master_encrypted_dek" binding:"required"`
	DEKNonce           string            `json:"dek_nonce" binding:"required"`
	Indexes            map[string]string `json:"indexes,omitempty"`
	Sensitivity        string            `json:"sensitivity,omitempty" binding:"omitempty,oneof=low medium high"`
}

type ScanRequest struct {
	TableHash string  `json:"table_hash" binding:"required"`
	Limit     int     `json:"limit,omitempty"`
	NextToken *string `json:"next_token,omitempty"`
}
