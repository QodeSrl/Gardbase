package objects

import "time"

type GetTableHashResponse struct {
	TableHash string `json:"table_hash"`
}

type GetTableIEKResponse struct {
	IEK []byte `json:"iek"`
}

type PutObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TableHash string    `json:"table_hash"`
	Version   int32     `json:"version"`
}

type RequestPutLargeObjectResponse struct {
	UploadURL       string `json:"upload_url"`
	ObjectID        string `json:"object_id"`
	ExpiresIn       int64  `json:"expires_in_seconds"`
	ExpectedVersion int32  `json:"expected_version"`
}

type ConfirmPutLargeObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TableHash string    `json:"table_hash"`
	Version   int32     `json:"version"`
}

type ResultObject struct {
	ObjectID         string    `json:"object_id"`
	GetURL           string    `json:"get_url"`
	EncryptedBlob    []byte    `json:"encrypted_blob,omitempty"`
	KMSWrappedDEK    []byte    `json:"kms_wrapped_dek,omitempty"`
	MasterWrappedDEK []byte    `json:"master_wrapped_dek,omitempty"`
	DEKNonce         []byte    `json:"dek_nonce,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int32     `json:"version"`
}

type GetObjectResponse = ResultObject

type ScanResponse struct {
	Objects   []ResultObject `json:"objects"`
	Count     int            `json:"count"`
	NextToken *string        `json:"next_token,omitempty"`
}

type QueryResponse struct {
	Objects   []ResultObject `json:"objects"`
	Count     int            `json:"count"`
	NextToken *string        `json:"next_token,omitempty"`
}
