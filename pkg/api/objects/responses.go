package objects

import "time"

type GetTableHashResponse struct {
	TableHash string `json:"table_hash"`
}

type PutObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int64     `json:"expires_in_seconds"`
	CreatedAt time.Time `json:"created_at"`
	TableHash string    `json:"table_hash"`
}

type ResultObject struct {
	ObjectID         string    `json:"object_id"`
	GetURL           string    `json:"get_url"`
	EncryptedBlob    string    `json:"encrypted_blob,omitempty"`
	KMSWrappedDEK    string    `json:"kms_wrapped_dek,omitempty"`
	MasterWrappedDEK string    `json:"master_wrapped_dek,omitempty"`
	DEKNonce         string    `json:"dek_nonce,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int32     `json:"version"`
}

type GetObjectResponse = ResultObject

type ScanResponse struct {
	Objects   []ResultObject `json:"objects"`
	NextToken *string        `json:"next_token,omitempty"`
}
