package models

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type Index struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>#IDX#<index_name>"
	SK []byte `dynamodbav:"sk" json:"sk"` // format: "<index_token>"

	GSI1PK string `dynamodbav:"gsi1pk,omitempty" json:"gsi1pk,omitempty"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>#OBJ#<object_id>"
	GSI1SK string `dynamodbav:"gsi1sk,omitempty" json:"gsi1sk,omitempty"` // format: "IDX#<index_name>"

	S3Key     string    `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"` // Duplicated S3 key for quick access (larger blobs)
	CreatedAt time.Time `dynamodbav:"created_at,omitempty" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at,omitempty" json:"updated_at"`
}

const (
	DETHashValueLength           = 32
	OPERangeValueLength          = 8 // TODO
	ObjectIDLength               = 16
	IndexTokenHashLength         = DETHashValueLength + ObjectIDLength
	IndexTokenHashAndRangeLength = DETHashValueLength + OPERangeValueLength + ObjectIDLength
)

var (
	MinRangeValue = make([]byte, OPERangeValueLength) // assuming 8-byte range values for OPE, adjust if needed (TODO)
	MaxRangeValue = bytes.Repeat([]byte{0xFF}, OPERangeValueLength)
)

func NewIndex(indexName string, tenantId string, tableHash string, indexToken []byte, objectId string, s3Key string) *Index {
	return &Index{
		PK:        GenerateIndexPK(tenantId, tableHash, indexName),
		SK:        indexToken,
		GSI1PK:    GenerateGSI1PK(tenantId, tableHash, objectId),
		GSI1SK:    GenerateGSI1SK(indexName),
		S3Key:     s3Key,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func GenerateIndexPK(tenantId string, tableHash string, indexName string) string {
	return fmt.Sprintf("TENANT#%s#TABLE#%s#IDX#%s", tenantId, tableHash, indexName)
}

func (i *Index) GetIndexName() string {
	// PK format: "TENANT#<tenant_id>#TABLE#<table_hash>#IDX#<index_name>"
	parts := strings.SplitN(i.PK, "#", 5)
	if len(parts) == 5 && parts[4] == "IDX" {
		return parts[5]
	}
	return ""
}

func (i *Index) GetTableHash() string {
	// PK format: "TENANT#<tenant_id>#TABLE#<table_hash>#IDX#<index_name>"
	parts := strings.SplitN(i.PK, "#", 5)
	if len(parts) == 5 && parts[2] == "TABLE" {
		return parts[3]
	}
	return ""
}

func (i *Index) GetObjectID() string {
	// GSI1PK format: "TENANT#<tenant_id>#TABLE#<table_hash>#OBJ#<object_id>"
	parts := strings.SplitN(i.GSI1PK, "#", 6)
	if len(parts) == 6 && parts[4] == "OBJ" {
		return parts[5]
	}
	return ""
}

func (i *Index) GetToken() []byte {
	return i.SK
}

func GenerateGSI1PK(tenantId string, tableHash string, objectId string) string {
	return fmt.Sprintf("TENANT#%s#TABLE#%s#OBJ#%s", tenantId, tableHash, objectId)
}

func GenerateGSI1SK(indexName string) string {
	return fmt.Sprintf("IDX#%s", indexName)
}
