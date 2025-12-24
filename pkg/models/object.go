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
	UpdatedAt time.Time `dynamodbav:"updated_at,omitempty" json:"updated_at,omitempty"`
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

func NewObject(tenantId string, objectId string, s3Key string, encryptedDek string) *Object {
	return &Object{
		PK:           fmt.Sprintf("TENANT#%s", tenantId),
		SK:           fmt.Sprintf("OBJ#%s", objectId),
		S3Key:        s3Key,
		EncryptedDEK: encryptedDek,
		Status:       StatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Version:      1,
	}
}

type CreateObjectRequest struct {
	EncryptedDEK string            `json:"encrypted_dek" binding:"required"`
	Indexes      map[string]string `json:"indexes,omitempty"`
	Sensitivity  string            `json:"sensitivity,omitempty" binding:"omitempty,oneof=low medium high"`
}

type CreateObjectResponse struct {
	ObjectID  string    `json:"object_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresIn int64     `json:"expires_in_seconds"`
	CreatedAt time.Time `json:"created_at"`
}

type GetObjectResponse struct {
	ObjectID     string    `json:"object_id"`
	GetURL       string    `json:"get_url"`
	EncryptedDEK string    `json:"encrypted_dek"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Version      int32     `json:"version"`
}
