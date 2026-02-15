package models

import (
	"fmt"
	"strings"
	"time"
)

type TableConfig struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>"

	// index encryption key
	KMSWrappedIEK string `dynamodbav:"wrapped_iek" json:"kms_wrapped_iek"`

	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func NewTableConfig(tenantId string, tableHash string, kmsWrappedIEK string) *TableConfig {
	return &TableConfig{
		PK:            GenerateTableConfigPK(tenantId, tableHash),
		KMSWrappedIEK: kmsWrappedIEK,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func GenerateTableConfigPK(tenantId string, tableHash string) string {
	return fmt.Sprintf("TENANT#%s#TABLE#%s", tenantId, tableHash)
}

func (t *TableConfig) GetTenantId() string {
	// PK format: "TENANT#<tenant_id>#TABLE#<table_hash>"
	parts := strings.SplitN(t.PK, "#", 4)
	if len(parts) == 4 && parts[0] == "TENANT" && parts[2] == "TABLE" {
		return parts[1]
	}
	return ""
}
