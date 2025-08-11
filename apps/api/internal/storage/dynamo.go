package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DynamoClient struct {
	Client *dynamodb.Client
	ObjectsTable string
	IndexesTable string
}

func NewDynamoClient(ctx context.Context, objectsTable string, indexesTable string, cfg aws.Config) *DynamoClient {
	return &DynamoClient{
		Client: dynamodb.NewFromConfig(cfg),
		ObjectsTable: objectsTable,
		IndexesTable: indexesTable,
	}
}
