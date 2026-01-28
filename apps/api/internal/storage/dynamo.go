package storage

import (
	"context"
	"fmt"

	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type DynamoClient struct {
	Client            *dynamodb.Client
	ObjectsTable      string
	IndexesTable      string
	TenantConfigTable string
	APIKeysTable      string
}

func NewDynamoClient(ctx context.Context, objectsTable string, indexesTable string, tenantConfigTable string, apiKeysTable string, cfg aws.Config, useLocalstack bool, localstackUrl string) *DynamoClient {
	return &DynamoClient{
		Client: dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			if useLocalstack {
				o.BaseEndpoint = aws.String(localstackUrl)
			}
		}),
		ObjectsTable:      objectsTable,
		IndexesTable:      indexesTable,
		TenantConfigTable: tenantConfigTable,
		APIKeysTable:      apiKeysTable,
	}
}

func (d *DynamoClient) TestConnnectivity(ctx context.Context) error {
	_, err := d.Client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.ObjectsTable),
	})
	return err
}

/*
CreateObjectWithIndexes stores the given object in DynamoDB and creates associated index entries.
It first marshals the object and index data into DynamoDB attribute maps.
If the total number of items to write (object + indexes) is 25 or fewer, it performs a single transactional write.
Otherwise, it writes the object separately and batches the index writes in groups of 25 using BatchWriteItem.
Returns an error if any DynamoDB operation fails.
*/
func (d *DynamoClient) CreateObjectWithIndexes(ctx context.Context, tableHash string, obj *models.Object, indexes map[string]string) error {
	objMap, err := attributevalue.MarshalMap(obj)
	if err != nil {
		return err
	}

	indexItems := make([]map[string]ddbtypes.AttributeValue, 0, len(indexes))
	for idxName, token := range indexes {
		index := models.NewIndex(idxName, extractTenantFromPK(obj.PK), tableHash, token, extractObjectIdFromSK(obj.SK), obj.S3Key)
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
				Item:      objMap,
			},
		})

		// indexes puts
		for _, item := range indexItems {
			twrite = append(twrite, ddbtypes.TransactWriteItem{
				Put: &ddbtypes.Put{
					TableName: aws.String(d.IndexesTable),
					Item:      item,
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
		Item:      objMap,
	})
	if err != nil {
		return err
	}

	const batchSize = 25
	for i := 0; i < len(indexItems); i += batchSize {
		end := min(i+batchSize, len(indexItems))

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
func (d *DynamoClient) GetObject(ctx context.Context, tenantId string, tableHash string, objectId string) (*models.Object, error) {
	pk := fmt.Sprintf("TENANT#%s#TABLE#%s", tenantId, tableHash)
	sk := fmt.Sprintf("OBJ#%s", objectId)

	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.ObjectsTable,
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberS{Value: sk},
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

func (d *DynamoClient) BatchGetEncryptedDEKs(ctx context.Context, tenantId string, tableHash string, objectsIds []string) (map[string]string, error) {
	if len(objectsIds) == 0 {
		return map[string]string{}, nil
	}
	keys := make([]map[string]ddbtypes.AttributeValue, 0, len(objectsIds))
	for _, objectId := range objectsIds {
		pk := fmt.Sprintf("TENANT#%s#TABLE#%s", tenantId, tableHash)
		sk := fmt.Sprintf("OBJ#%s", objectId)
		keys = append(keys, map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberS{Value: sk},
		})
	}

	out, err := d.Client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]ddbtypes.KeysAndAttributes{
			d.ObjectsTable: {
				Keys:                 keys,
				ProjectionExpression: aws.String("sk, kms_wrapped_dek"),
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
		result[extractObjectIdFromSK(obj.SK)] = obj.KMSWrappedDEK
	}
	return result, nil
}

func (d *DynamoClient) CreateTenant(ctx context.Context, tenantID string, wrappedMasterKey []byte, wrappedTableSalt []byte) error {
	tenantConfig := models.NewTenantConfig(tenantID, wrappedMasterKey, wrappedTableSalt, 1)
	item, err := attributevalue.MarshalMap(tenantConfig)
	if err != nil {
		return err
	}
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.TenantConfigTable),
		Item:      item,
	})
	return err
}

func (d *DynamoClient) GetTenant(ctx context.Context, tenantID string) (*models.TenantConfig, error) {
	pk := fmt.Sprintf("TENANT#%s", tenantID)

	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.TenantConfigTable,
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var tenantConfig models.TenantConfig
	if err := attributevalue.UnmarshalMap(out.Item, &tenantConfig); err != nil {
		return nil, err
	}
	return &tenantConfig, nil
}

func (d *DynamoClient) CreateAPIKey(ctx context.Context, tenantID string) (string, error) {
	apiKey := models.GenerateAPIKey()
	hashedKey, err := models.HashAPIKey(apiKey)
	if err != nil {
		return "", err
	}
	apiKeyModel := models.NewAPIKey(tenantID, uuid.NewString(), hashedKey, []string{"read", "write"}, nil)
	item, err := attributevalue.MarshalMap(apiKeyModel)
	if err != nil {
		return "", err
	}
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.TenantConfigTable),
		Item:      item,
	})
	return apiKey, err
}

// Helper function to extract tenant from PK of format "TENANT#<tenant_id>#TABLE#<table_hash>"
func extractTenantFromPK(pk string) string {
	var tenantId string
	fmt.Sscanf(pk, "TENANT#%s#TABLE#", &tenantId)
	return tenantId
}

// Helper function to extract object ID from SK of format "OBJ#<object_id>"
func extractObjectIdFromSK(sk string) string {
	var objectId string
	fmt.Sscanf(sk, "OBJ#%s", &objectId)
	return objectId
}
