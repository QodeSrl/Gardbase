package objects

type GetTableHashRequest struct {
	SessionID                 string `json:"session_id" binding:"required"` // Base64-encoded session ID
	SessionEncryptedTableName string `json:"encrypted_table_name,omitempty"`
	SessionTableNameNonce     string `json:"table_name_nonce,omitempty"`
}

type GetTableIEKRequest struct {
	SessionID string `json:"session_id" binding:"required"` // Base64-encoded session ID
	TableHash string `json:"table_hash" binding:"required"` // Base64-encoded table hash
}

type IndexName struct {
	HashField  string  `json:"hash_field" binding:"required"`
	RangeField *string `json:"range_field,omitempty"`
}

type Index struct {
	Name  IndexName `json:"name"`
	Token []byte    `json:"token" binding:"required"`
}

// If the object is lightweight (e.g. encrypted blob is less than 100KB), the client can include the encrypted blob and DEK in the request body to avoid an extra round trip for uploading the object.
type PutObjectRequest struct {
	ObjectID           string  `json:"object_id,omitempty"` // Optional for updates, auto-generated for new objects
	TableHash          string  `json:"table_hash" binding:"required"`
	EncryptedBlob      string  `json:"encrypted_blob" binding:"required"`
	KMSEncryptedDEK    string  `json:"encrypted_dek" binding:"required"`
	MasterEncryptedDEK string  `json:"master_encrypted_dek" binding:"required"`
	DEKNonce           string  `json:"dek_nonce" binding:"required"`
	Indexes            []Index `json:"indexes,omitempty"`
	Sensitivity        string  `json:"sensitivity,omitempty" binding:"omitempty,oneof=low medium high"`
	Version            int32   `json:"version,omitempty"` // 1 = new object, >1 = update
}

// For large objects, the client should first call RequestPutLargeObject to get a pre-signed URL for uploading the encrypted blob, then call ConfirmPutLargeObject to create the object record after the upload is complete.
type RequestPutLargeObjectRequest struct {
	ObjectID  string `json:"object_id,omitempty"` // Optional for updates, auto-generated for new objects
	TableHash string `json:"table_hash" binding:"required"`
	BlobSize  int64  `json:"blob_size" binding:"required"`
	Version   int32  `json:"version,omitempty"` // 1 = new object, >1 = update
}

type ConfirmPutLargeObjectRequest struct {
	ObjectID           string  `json:"object_id" binding:"required"`
	TableHash          string  `json:"table_hash" binding:"required"`
	KMSEncryptedDEK    string  `json:"encrypted_dek" binding:"required"`
	MasterEncryptedDEK string  `json:"master_encrypted_dek" binding:"required"`
	DEKNonce           string  `json:"dek_nonce" binding:"required"`
	Indexes            []Index `json:"indexes,omitempty"`
	Sensitivity        string  `json:"sensitivity,omitempty" binding:"omitempty,oneof=low medium high"`
	Version            int32   `json:"version,omitempty"` // 1 = new object, >1 = update
}

type GetObjectRequest struct {
	TableHash string `json:"table_hash" binding:"required"`
	ObjectID  string `json:"object_id" binding:"required"`
}

type ScanRequest struct {
	TableHash string  `json:"table_hash" binding:"required"`
	Limit     int     `json:"limit,omitempty"`
	NextToken *string `json:"next_token,omitempty"`
}
