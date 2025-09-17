package storage

import (
	"context"
	"fmt"

	"github.com/QodeSrl/gardbase/pkg/models"
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

func NewDynamoClient(ctx context.Context, objectsTable string, indexesTable string, cfg aws.Config, useLocalstack bool, localstackUrl string) *DynamoClient {
	return &DynamoClient{
		Client: dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			if useLocalstack {
				o.BaseEndpoint = aws.String(localstackUrl)
			}
		}),
		ObjectsTable: objectsTable,
		IndexesTable: indexesTable,
	}
}

/* 
   CreateObjectWithIndexes stores the given object in DynamoDB and creates associated index entries.
   It first marshals the object and index data into DynamoDB attribute maps.
   If the total number of items to write (object + indexes) is 25 or fewer, it performs a single transactional write.
   Otherwise, it writes the object separately and batches the index writes in groups of 25 using BatchWriteItem.
   Returns an error if any DynamoDB operation fails.
*/
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

/*
   GetObject retrieves an object by tenant ID and object ID from DynamoDB.
*/
func (d *DynamoClient) GetObject(ctx context.Context, tenantId string, objectId string) (*models.Object, error) {
	pk := fmt.Sprintf("TENANT#%s", tenantId)
	sk := fmt.Sprintf("OBJ#%s", objectId)

	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.ObjectsTable,
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{ Value: pk },
			"sk": &ddbtypes.AttributeValueMemberS{ Value: sk },
		},
	})

	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}

	var obj models.Object
	if err := attributevalue.UnmarshalMap(out.Item, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}

func (d *DynamoClient) BatchGetEncryptedDEKs(ctx context.Context, tenantId string, objectsIds []string) (map[string]string, error) {
	if len(objectsIds) == 0 {
		return map[string]string{}, nil
	}
	keys := make([]map[string]ddbtypes.AttributeValue, 0, len(objectsIds))
	for _, objectId := range objectsIds {
		pk := fmt.Sprintf("TENANT#%s", tenantId)
		sk := fmt.Sprintf("OBJ#%s", objectId)
		keys = append(keys, map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{ Value: pk },
			"sk": &ddbtypes.AttributeValueMemberS{ Value: sk },
		})
	}

	out, err := d.Client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]ddbtypes.KeysAndAttributes{
			d.ObjectsTable: {
				Keys: keys,
				ProjectionExpression: aws.String("sk, encrypted_dek"),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(objectsIds))
	if out.Responses == nil || out.Responses[d.ObjectsTable] == nil {
		return result, nil
	}
	for _, item := range out.Responses[d.ObjectsTable] {
		var obj models.Object
		if err := attributevalue.UnmarshalMap(item, &obj); err != nil {
			return nil, err
		}
		result[extractObjectIdFromSK(obj.SK)] = obj.EncryptedDEK
	}
	return result, nil
}

// Helper function to extract tenant from PK of format "TENANT#<tenant_id>"
func extractTenantFromPK(pk string) string {
	var tenantId string
	fmt.Sscanf(pk, "TENANT#%s", &tenantId)
	return tenantId
}

// Helper function to extract object ID from SK of format "OBJ#<object_id>"
func extractObjectIdFromSK(sk string) string {
	var objectId string
	fmt.Sscanf(sk, "OBJ#%s", &objectId)
	return objectId
}
