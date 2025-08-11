package models

import (
	"fmt"
	"time"
)

type Index struct {
	PK string `dynamodbav:"pk" json:"pk"` // format: "IDX#<index_name>#TENANT#<tenant_id>"
	SK string `dynamodbav:"sk" json:"sk"` // format: "TOKEN#<index_token>#OBJ#<object_id>"

	ObjectID  string `dynamodbav:"object_id,omitempty" json:"object_id,omitempty"`
	S3Key     string `dynamodbav:"s3_key,omitempty" json:"s3_key,omitempty"` // Duplicated S3 key for quick access
	CreatedAt time.Time `dynamodbav:"created_at,omitempty" json:"created_at,omitempty"`
}

func NewIndex(indexName string, tenantId string, indexToken string, objectId string, s3Key string) *Index {
	return &Index{
		PK:        fmt.Sprintf("IDX#%s#TENANT#%s", indexName, tenantId),
		SK:        fmt.Sprintf("TOKEN#%s#OBJ#%s", indexToken, objectId),
		S3Key:     s3Key,
		ObjectID:  objectId,
		CreatedAt: time.Now().UTC(),
	}
}