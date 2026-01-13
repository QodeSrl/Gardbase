package models

import "time"

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
		WrappedMasterKey: string(wrappedMasterKKey),
		WrappedTableSalt: string(wrappedTableSalt),
		MasterKeyVersion: masterKeyVersion,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

type CreateTenantRequest struct {
	ClientPubKey string `json:"client_public_key" binding:"required"`
}

type CreateTenantResponse struct {
	TenantID            string `json:"tenant_id"`
	EncryptedMasterKey  string `json:"encrypted_master_key"`
	EncryptedTableSalt  string `json:"encrypted_table_salt"`
	EnclavePubKey       string `json:"enclave_public_key"`
	AttestationDocument string `json:"attestation_document"`
	Nonce               string `json:"nonce"`
	APIKey              string `json:"api_key"`
}
