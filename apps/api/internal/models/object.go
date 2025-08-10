package models

import (
	"fmt"
	"time"
)

type Object struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>"
	SK string `dynamodbav:"sk" json:"sk"` // format: "OBJ#<object_id>"

	S3Key        string `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"`
	EncryptedDEK string `dynamodbav:"encrypted_dek,omitempty" json:"encrypted_dek,omitempty"`
	Sensitivity  string `dynamodbav:"sensitivity,omitempty" json:"sensitivity,omitempty"`

	CreatedAt time.Time `dynamodbav:"created_at,omitempty" json:"created_at,omitempty"`
	Version   int32  `dynamodbav:"version,omitempty" json:"version,omitempty"`
	Status    string `dynamodbav:"status,omitempty" json:"status,omitempty"` // "pending", "ready", "deleted"
	TTL       int64  `dynamodbav:"ttl,omitempty" json:"ttl,omitempty"`       // Unix timestamp for expiration
}

const (
	StatusPending = "pending"
	StatusReady   = "ready"
	StatusDeleted = "deleted"
)

func NewObject(tenantId string, objectId string) *Object {
	return &Object{
		PK: fmt.Sprintf("TENANT#%s", tenantId),
		SK: fmt.Sprintf("OBJ#%s", objectId),
		CreatedAt: time.Now().UTC(),
		Version:   1,
	}
}