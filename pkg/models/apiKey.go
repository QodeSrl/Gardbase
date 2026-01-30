package models

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type APIKey struct {
	PK string `dynamodbav:"pk" json:"pk"` // "TENANT#<tenant_id>"
	SK string `dynamodbav:"sk" json:"sk"` // "APIKEY#<key_id>"

	HashedKey   string     `dynamodbav:"hashed_key" json:"hashed_key"`
	Permissions []string   `dynamodbav:"permissions" json:"permissions"`
	CreatedAt   time.Time  `dynamodbav:"created_at" json:"created_at"`
	ExpiresAt   *time.Time `dynamodbav:"expires_at" json:"expires_at"`
}

const (
	PermissionRead  = "read"
	PermissionWrite = "write"
)

func NewAPIKey(tenantId string, keyId string, hashedKey string, permissions []string, expiresAt *time.Time) *APIKey {
	return &APIKey{
		PK:          "TENANT#" + tenantId,
		SK:          "APIKEY#" + keyId,
		HashedKey:   hashedKey,
		Permissions: permissions,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}
}

func GenerateAPIKey() string {
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	apiKey := "gdb_live_" + base64.URLEncoding.EncodeToString(randomBytes)
	return apiKey
}

func HashAPIKey(apiKey string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	return string(hashedBytes), err
}
