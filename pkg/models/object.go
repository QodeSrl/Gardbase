package models

import (
	"fmt"
	"time"
)

type Object struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>"
	SK string `dynamodbav:"sk" json:"sk"` // format: "OBJ#<object_id>"

	EncryptedBlob string `dynamodbav:"encrypted_blob,omitempty" json:"encrypted_blob,omitempty"` // < 100KB blobs stored inline
	S3Key         string `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"`                 // S3 object key for larger blobs

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

func NewObject(tenantId string, tableHash string, objectId string, kmsWrappedDEK string, masterWrappedDEK string, dekNonce string) *Object {
	return &Object{
		PK:               fmt.Sprintf("TENANT#%s#TABLE#%s", tenantId, tableHash),
		SK:               fmt.Sprintf("OBJ#%s", objectId),
		KMSWrappedDEK:    kmsWrappedDEK,
		MasterWrappedDEK: masterWrappedDEK,
		DEKNonce:         dekNonce,
		Status:           StatusPending,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Version:          1,
	}
}
