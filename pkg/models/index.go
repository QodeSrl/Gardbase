package models

import (
	"fmt"
	"time"
)

type Index struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "TENANT#<tenant_id>#TABLE#<table_hash>#IDX#<index_name>"
	SK string `dynamodbav:"sk" json:"sk"` // format: "TOKEN#<index_token>#OBJ#<object_id>"

	S3Key     string    `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"` // Duplicated S3 key for quick access
	CreatedAt time.Time `dynamodbav:"created_at,omitempty" json:"created_at"`
}

func NewIndex(indexName string, tenantId string, tableHash string, indexToken string, objectId string, s3Key string) *Index {
	return &Index{
		PK:        fmt.Sprintf("TENANT#%s#TABLE#%s#IDX#%s", tenantId, tableHash, indexName),
		SK:        fmt.Sprintf("TOKEN#%s#OBJ#%s", indexToken, objectId),
		S3Key:     s3Key,
		CreatedAt: time.Now().UTC(),
	}
}
