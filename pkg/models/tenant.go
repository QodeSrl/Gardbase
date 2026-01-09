package models

import "time"

type TenantConfig struct {
	PK string `dynamodbav:"pk"` // "TENANT#<tenant_id>"
	SK string `dynamodbav:"sk"` // "#CONFIG"

	// Wrapped keys encrypted with KMS
	WrappedMasterKey string `dynamodbav:"wrapped_master_key"`
	WrappedTableSalt string `dynamodbav:"wrapped_table_salt"`

	// Key metadata
	MasterKeyVersion int       `dynamodbav:"master_key_version"`
	MasterKeyID      string    `dynamodbav:"master_key_id"`
	CreatedAt        time.Time `dynamodbav:"created_at"`
	UpdatedAt        time.Time `dynamodbav:"updated_at"`

	PreviousWrappedKeys []HistoricalKey `dynamodbav:"previous_wrapped_keys,omitempty"`
	LastRotatedAt       *time.Time      `dynamodbav:"last_rotated_at,omitempty"`

	RecoveryEmail    string   `dynamodbav:"recovery_email,omitempty"`
	RecoveryContacts []string `dynamodbav:"recovery_contacts,omitempty"`
}

type HistoricalKey struct {
	WrappedKey string    `dynamodbav:"wrapped_key"`
	Version    string    `dynamodbav:"version"`
	ValidUntil time.Time `dynamodbav:"valid_until"`
}

func NewTenantConfig(tenantID string, wrappedMasterKKey []byte, wrappedTableSalt []byte, masterKeyVersion int, masterKeyID string) *TenantConfig {
	return &TenantConfig{
		PK:               "TENANT#" + tenantID,
		SK:               "#CONFIG",
		WrappedMasterKey: string(wrappedMasterKKey),
		WrappedTableSalt: string(wrappedTableSalt),
		MasterKeyVersion: masterKeyVersion,
		MasterKeyID:      masterKeyID,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}
