package models

import (
	"encoding/base64"
	"time"
)

type TenantConfig struct {
	PK string `dynamodbav:"pk"` // "TENANT#<tenant_id>"

	// Wrapped keys encrypted with KMS
	WrappedMasterKey string `dynamodbav:"wrapped_master_key" json:"wrapped_master_key"`
	WrappedTableSalt string `dynamodbav:"wrapped_table_salt" json:"wrapped_table_salt"`

	// Key metadata
	MasterKeyVersion int `dynamodbav:"master_key_version" json:"master_key_version"`

	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func NewTenantConfig(tenantID string, wrappedMasterKKey []byte, wrappedTableSalt []byte, masterKeyVersion int) *TenantConfig {
	return &TenantConfig{
		PK:               "TENANT#" + tenantID,
		WrappedMasterKey: base64.StdEncoding.EncodeToString(wrappedMasterKKey),
		WrappedTableSalt: base64.StdEncoding.EncodeToString(wrappedTableSalt),
		MasterKeyVersion: masterKeyVersion,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}
