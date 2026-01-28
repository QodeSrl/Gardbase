package models

import (
	"fmt"
	"time"
)

type Object struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>"
	SK string `dynamodbav:"sk" json:"sk"` // format: "OBJ#<object_id>"

	S3Key string `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"`

	KMSWrappedDEK    string `dynamodbav:"kms_wrapped_dek,omitempty" json:"kms_wrapped_dek,omitempty"`       // DEK wrapped with KMS
	MasterWrappedDEK string `dynamodbav:"master_wrapped_dek,omitempty" json:"master_wrapped_dek,omitempty"` // DEK wrapped with tenant master key
	DEKNonce         string `dynamodbav:"dek_nonce,omitempty" json:"dek_nonce,omitempty"`                   // Nonce used for master_wrapped_dek

	Sensitivity string `dynamodbav:"sensitivity,omitempty" json:"sensitivity,omitempty"` // TODO: this is still unused

	CreatedAt time.Time `dynamodbav:"created_at,omitempty" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at,omitempty" json:"updated_at"`
	Version   int32     `dynamodbav:"version,omitempty" json:"version,omitempty"`
	Status    string    `dynamodbav:"status,omitempty" json:"status,omitempty"` // "pending", "ready", "deleted"
	TTL       int64     `dynamodbav:"ttl,omitempty" json:"ttl,omitempty"`       // Unix timestamp for expiration
}

const (
	StatusPending = "pending"
	StatusReady   = "ready"
	StatusDeleted = "deleted"
)

const (
	SensitivityLow    = "low"
	SensitivityMedium = "medium"
	SensitivityHigh   = "high"
)

func NewObject(tenantId string, tableHash string, objectId string, s3Key string, kmsWrappedDEK string, masterWrappedDEK string, dekNonce string) *Object {
	return &Object{
		PK:               fmt.Sprintf("TENANT#%s#TABLE#%s", tenantId, tableHash),
		SK:               fmt.Sprintf("OBJ#%s", objectId),
		S3Key:            s3Key,
		KMSWrappedDEK:    kmsWrappedDEK,
		MasterWrappedDEK: masterWrappedDEK,
		DEKNonce:         dekNonce,
		Status:           StatusPending,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Version:          1,
	}
}

type GetTableHashRequest struct {
	SessionID                 string `json:"session_id" binding:"required"` // Base64-encoded session ID
	SessionEncryptedTableName string `json:"encrypted_table_name,omitempty"`
	SessionTableNameNonce     string `json:"table_name_nonce,omitempty"`
}

type GetTableHashResponse struct {
	TableHash string `json:"table_hash"`
}

type CreateObjectRequest struct {
	KMSEncryptedDEK    string            `json:"encrypted_dek" binding:"required"`
	MasterEncryptedDEK string            `json:"master_encrypted_dek" binding:"required"`
	DEKNonce           string            `json:"dek_nonce" binding:"required"`
	TableHash          string            `json:"table_hash" binding:"required"`
	Indexes            map[string]string `json:"indexes,omitempty"`
	Sensitivity        string            `json:"sensitivity,omitempty" binding:"omitempty,oneof=low medium high"`
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
