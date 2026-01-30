package objects

import "time"

type GetTableHashResponse struct {
	TableHash string `json:"table_hash"`
}

type CreateObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int64     `json:"expires_in_seconds"`
	CreatedAt time.Time `json:"created_at"`
	TableHash string    `json:"table_hash"`
}

type GetObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	GetURL    string    `json:"get_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int32     `json:"version"`
}
