package storage

import (
	"context"
	"fmt"

	"github.com/QodeSrl/gardbase-api/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func (d *DynamoClient) CreateObjectWithIndexes(ctx context.Context, obj *models.Object, indexes map[string]string) error {
	objMap, err := attributevalue.MarshalMap(obj)
	if err != nil {
		return err
	}

	indexItems := make([]map[string]ddbtypes.AttributeValue, 0, len(indexes))
	for idxName, token := range indexes {
		index := models.NewIndex(idxName, extractTenantFromPK(obj.PK), token, extractObjectIdFromSK(obj.SK), obj.S3Key)
		av, err := attributevalue.MarshalMap(index)
		if err != nil {
			return err
		}
		indexItems = append(indexItems, av)
	}

	// if total items to put is <= 25, use a single transact write
	if len(indexItems) <= 25 {
		twrite := make([]ddbtypes.TransactWriteItem, 0, len(indexItems)+1)

		// obj put
		twrite = append(twrite, ddbtypes.TransactWriteItem{
			Put: &ddbtypes.Put{
				TableName: aws.String(d.ObjectsTable),
				Item: objMap,
			},
		})

		// indexes puts
		for _, item := range indexItems {
			twrite = append(twrite, ddbtypes.TransactWriteItem{
				Put: &ddbtypes.Put{
					TableName: aws.String(d.IndexesTable),
					Item: item,
				},
			})
		}

		_, err = d.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: twrite,
		})
		return err
	}

	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.ObjectsTable),
		Item: objMap,
	})
	if err != nil {
		return err
	}

	const batchSize = 25
	for i := 0; i < len(indexItems); i += batchSize {
		end := min(i + batchSize, len(indexItems))

		writeRequests := make([]ddbtypes.WriteRequest, 0, end-i)
		for _, item := range indexItems[i:end] {
			writeRequests = append(writeRequests, ddbtypes.WriteRequest{
				PutRequest: &ddbtypes.PutRequest{
					Item: item,
				},
			})
		}

		_, err = d.Client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbtypes.WriteRequest{
				d.IndexesTable: writeRequests,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// extract tenant from PK of format "TENANT#<tenant_id>"
func extractTenantFromPK(pk string) string {
	var tenantId string
	fmt.Sscanf(pk, "TENANT#%s", &tenantId)
	return tenantId
}

// extract object ID from SK of format "OBJ#<object_id>"
func extractObjectIdFromSK(sk string) string {
	var objectId string
	fmt.Sscanf(sk, "OBJ#%s", &objectId)
	return objectId
}