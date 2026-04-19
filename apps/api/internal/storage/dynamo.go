package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QodeSrl/gardbase/pkg/api/objects"
	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type DynamoClient struct {
	Client            *dynamodb.Client
	ObjectsTable      string
	IndexesTable      string
	TableConfigTable  string
	TenantConfigTable string
	APIKeysTable      string
}

func NewDynamoClient(ctx context.Context, objectsTable string, indexesTable string, tableConfigTable string, tenantConfigTable string, apiKeysTable string, cfg aws.Config, useLocalstack bool, localstackUrl string) *DynamoClient {
	return &DynamoClient{
		Client: dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			if useLocalstack {
				o.BaseEndpoint = aws.String(localstackUrl)
			}
		}),
		ObjectsTable:      objectsTable,
		IndexesTable:      indexesTable,
		TableConfigTable:  tableConfigTable,
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

func (d *DynamoClient) GetWrappedTableIEK(ctx context.Context, tenantId string, tableHash string) ([]byte, error) {
	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.TableConfigTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: models.GenerateTableConfigPK(tenantId, tableHash)},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var tableConfig models.TableConfig
	if err := attributevalue.UnmarshalMap(out.Item, &tableConfig); err != nil {
		return nil, err
	}
	return tableConfig.KMSWrappedIEK, nil
}

func (d *DynamoClient) SetWrappedTableIEK(ctx context.Context, tenantId string, tableHash string, kmsWrappedIEK []byte) error {
	tableConfig := models.NewTableConfig(tenantId, tableHash, kmsWrappedIEK)
	item, err := attributevalue.MarshalMap(tableConfig)
	if err != nil {
		return err
	}
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.TableConfigTable),
		Item:      item,
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
func (d *DynamoClient) CreateObjectWithIndexes(ctx context.Context, tableHash string, obj *models.Object, indexes []objects.Index) error {
	objMap, err := attributevalue.MarshalMap(obj)
	if err != nil {
		return err
	}

	indexItems := make([]map[string]ddbTypes.AttributeValue, 0, len(indexes))
	for _, idx := range indexes {
		objIdBytes, err := uuid.Parse(obj.GetObjectID())
		if err != nil {
			return fmt.Errorf("failed to parse object ID as UUID: %v", err)
		}
		token := append(idx.GetIndexToken(), objIdBytes[:16]...) // append object ID to the index token to ensure uniqueness across objects with the same index values
		index := models.NewIndex(idx.GetIndexName(), obj.GetTenantID(), tableHash, token, obj.GetObjectID(), obj.S3Key)
		av, err := attributevalue.MarshalMap(index)
		if err != nil {
			return err
		}
		indexItems = append(indexItems, av)
	}

	// if total items to put is <= 25, use a single transact write
	if len(indexItems) <= 25 {
		twrite := make([]ddbTypes.TransactWriteItem, 0, len(indexItems)+1)

		// obj put
		twrite = append(twrite, ddbTypes.TransactWriteItem{
			Put: &ddbTypes.Put{
				TableName: aws.String(d.ObjectsTable),
				Item:      objMap,
				ConditionExpression: aws.String(
					"attribute_not_exists(pk) AND attribute_not_exists(sk)",
				),
			},
		})

		// indexes puts
		for _, item := range indexItems {
			twrite = append(twrite, ddbTypes.TransactWriteItem{
				Put: &ddbTypes.Put{
					TableName: aws.String(d.IndexesTable),
					Item:      item,
					ConditionExpression: aws.String(
						"attribute_not_exists(pk) AND attribute_not_exists(sk)",
					),
				},
			})
		}

		_, err = d.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: twrite,
		})
		if err != nil {
			var condErr *ddbTypes.ConditionalCheckFailedException
			if errors.As(err, &condErr) {
				return ErrAlreadyExists
			}
		}
		return err
	}

	// if more than 25 items, write the object first, then batch write the indexes

	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.ObjectsTable),
		Item:      objMap,
		ConditionExpression: aws.String(
			"attribute_not_exists(pk) AND attribute_not_exists(sk)",
		),
	})
	if err != nil {
		var condErr *ddbTypes.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return ErrAlreadyExists
		}
		return err
	}

	const batchSize = 25
	for i := 0; i < len(indexItems); i += batchSize {
		end := min(i+batchSize, len(indexItems))

		writeRequests := make([]ddbTypes.WriteRequest, 0, end-i)
		for _, item := range indexItems[i:end] {
			writeRequests = append(writeRequests, ddbTypes.WriteRequest{
				PutRequest: &ddbTypes.PutRequest{
					Item: item,
				},
			})
		}

		_, err = d.Client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbTypes.WriteRequest{
				d.IndexesTable: writeRequests,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DynamoClient) UpdateObjectWithIndexes(ctx context.Context, tenantId string, tableHash string, objectId string, currentVersion int32, applyFn func(*models.Object), indexes []objects.Index) (*models.Object, error) {
	obj, err := d.GetObject(ctx, tenantId, tableHash, objectId)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, ErrNotFound
	}
	if obj.Status == models.StatusDeleted {
		return nil, ErrNotFoundOrDeleted
	}
	if obj.Version != currentVersion {
		return nil, ErrVersionMismatch
	}

	applyFn(obj)

	item, _ := attributevalue.MarshalMap(obj)
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.ObjectsTable),
		Item:      item,
		ConditionExpression: aws.String(
			"attribute_exists(pk) AND attribute_exists(sk) AND #v = :current AND #s <> :deleted",
		),
		ExpressionAttributeNames: map[string]string{
			"#v": "version",
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":current": &ddbTypes.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", currentVersion),
			},
			":deleted": &ddbTypes.AttributeValueMemberS{Value: models.StatusDeleted},
		},
	})
	if err != nil {
		var condErr *ddbTypes.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil, ErrVersionMismatch
		}
		return nil, err
	}

	if err := d.updateIndexes(ctx, tenantId, tableHash, objectId, indexes, obj.S3Key); err != nil {
		return nil, err
	}

	return obj, nil
}

func (d *DynamoClient) updateIndexes(ctx context.Context, tenantId string, tableHash string, objectId string, indexes []objects.Index, s3Key string) error {
	currentIndexes, err := d.GetIndexesByObjectID(ctx, tenantId, tableHash, objectId)
	if err != nil {
		return err
	}
	indexMap := make(map[string][]byte)
	for _, idx := range indexes {
		var name string
		if idx.Name.RangeField != nil {
			name = fmt.Sprintf("%s:%s", idx.Name.HashField, *idx.Name.RangeField)
		} else {
			name = idx.Name.HashField
		}
		objIdBytes, err := uuid.Parse(objectId)
		if err != nil {
			return fmt.Errorf("failed to parse object ID as UUID: %v", err)
		}
		indexMap[name] = append(idx.GetIndexToken(), objIdBytes[:16]...) // append object ID to the index token to ensure uniqueness across objects with the same index values
	}

	for _, idx := range currentIndexes {
		newIdx, exists := indexMap[idx.GetIndexName()]

		// if index doesn't exist in the new set, delete it
		if !exists {
			_, err := d.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(d.IndexesTable),
				Key: map[string]ddbTypes.AttributeValue{
					"pk": &ddbTypes.AttributeValueMemberS{Value: idx.PK},
					"sk": &ddbTypes.AttributeValueMemberB{Value: idx.SK},
				},
			})
			if err != nil {
				return err
			}
			continue
		}

		// if index exists but token has changed, update it
		if !bytes.Equal(newIdx, idx.GetToken()) {
			_, err := d.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String(d.IndexesTable),
				Key: map[string]ddbTypes.AttributeValue{
					"pk": &ddbTypes.AttributeValueMemberS{Value: idx.PK},
					"sk": &ddbTypes.AttributeValueMemberB{Value: idx.SK},
				},
				UpdateExpression: aws.String("SET sk = :newToken, updated_at = :updatedAt"),
				ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
					":newToken":  &ddbTypes.AttributeValueMemberB{Value: newIdx},
					":updatedAt": &ddbTypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
				},
			})
			if err != nil {
				return err
			}
		}
	}

	// add new indexes that didn't exist before
	for _, idx := range indexes {
		var idxName string
		if idx.Name.RangeField != nil {
			idxName = fmt.Sprintf("%s:%s", idx.Name.HashField, *idx.Name.RangeField)
		} else {
			idxName = idx.Name.HashField
		}
		if _, exists := currentIndexes[idxName]; exists {
			continue
		}
		objIdBytes, err := uuid.Parse(objectId)
		if err != nil {
			return fmt.Errorf("failed to parse object ID as UUID: %v", err)
		}
		newIndex := models.NewIndex(idxName, tenantId, tableHash, append(idx.GetIndexToken(), objIdBytes[:16]...), objectId, s3Key)
		item, err := attributevalue.MarshalMap(newIndex)
		if err != nil {
			return err
		}
		_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(d.IndexesTable),
			Item:      item,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DynamoClient) GetIndexesByObjectID(ctx context.Context, tenantId string, tableHash string, objectId string) (map[string]models.Index, error) {
	pk := models.GenerateGSI1PK(tenantId, tableHash, objectId)
	out, err := d.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.IndexesTable),
		IndexName:              aws.String("gsi1"),
		KeyConditionExpression: aws.String("gsi1pk = :pk"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk": &ddbTypes.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return nil, err
	}
	indexes := make(map[string]models.Index, len(out.Items))
	for _, item := range out.Items {
		var index models.Index
		if err := attributevalue.UnmarshalMap(item, &index); err != nil {
			return nil, err
		}
		indexes[index.GetIndexName()] = index
	}
	return indexes, nil
}

/*
GetObject retrieves an object by tenant ID and object ID from DynamoDB.
*/
func (d *DynamoClient) GetObject(ctx context.Context, tenantId string, tableHash string, objectId string) (*models.Object, error) {
	pk := models.GenerateObjectPK(tenantId, tableHash)
	sk := models.GenerateObjectSK(objectId)

	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.ObjectsTable,
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbTypes.AttributeValueMemberS{Value: sk},
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

func (d *DynamoClient) BatchGetEncryptedDEKs(ctx context.Context, tenantId string, tableHash string, objectsIds []string) (map[string][]byte, error) {
	if len(objectsIds) == 0 {
		return map[string][]byte{}, nil
	}
	keys := make([]map[string]ddbTypes.AttributeValue, 0, len(objectsIds))
	for _, objectId := range objectsIds {
		pk := models.GenerateObjectPK(tenantId, tableHash)
		sk := models.GenerateObjectSK(objectId)
		keys = append(keys, map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbTypes.AttributeValueMemberS{Value: sk},
		})
	}

	out, err := d.Client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]ddbTypes.KeysAndAttributes{
			d.ObjectsTable: {
				Keys:                 keys,
				ProjectionExpression: aws.String("sk, kms_wrapped_dek"),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte, len(objectsIds))
	if out.Responses == nil || out.Responses[d.ObjectsTable] == nil {
		return result, nil
	}
	for _, item := range out.Responses[d.ObjectsTable] {
		var obj models.Object
		if err := attributevalue.UnmarshalMap(item, &obj); err != nil {
			return nil, err
		}
		result[obj.GetObjectID()] = obj.KMSWrappedDEK
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
	pk := models.GenerateTenantConfigPK(tenantID)

	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.TenantConfigTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
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
	apiKeyModel := models.NewAPIKey(tenantID, uuid.NewString(), hashedKey, []string{models.PermissionRead, models.PermissionWrite}, nil)
	item, err := attributevalue.MarshalMap(apiKeyModel)
	if err != nil {
		return "", err
	}
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.APIKeysTable),
		Item:      item,
	})
	return apiKey, err
}

func (d *DynamoClient) FindAPIKey(ctx context.Context, tenantId string, providedKey string) (*models.APIKey, error) {
	pk := models.GenerateAPIKeyPK(tenantId)
	out, err := d.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.APIKeysTable),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk": &ddbTypes.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, nil
	}
	for _, item := range out.Items {
		var apiKey models.APIKey
		if err := attributevalue.UnmarshalMap(item, &apiKey); err != nil {
			return nil, err
		}
		err := bcrypt.CompareHashAndPassword([]byte(apiKey.HashedKey), []byte(providedKey))
		if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
			return nil, fmt.Errorf("API key expired")
		}
		if err == nil {
			return &apiKey, nil
		}
	}
	return nil, nil
}

type QueryResult struct {
	Objects   []models.Object
	Count     int
	NextToken *string
}

type ScanResult = QueryResult

func (d *DynamoClient) ScanTable(ctx context.Context, tenantID string, tableHash string, limit int, nextToken string) (*ScanResult, error) {
	var dynamoLimit *int32
	if limit > 0 {
		l := int32(limit)
		dynamoLimit = &l
	}
	pk := models.GenerateObjectPK(tenantID, tableHash)
	out, err := d.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.ObjectsTable),
		KeyConditionExpression: aws.String("pk = :pk"),
		FilterExpression:       aws.String("#status <> :deleted"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk":      &ddbTypes.AttributeValueMemberS{Value: pk},
			":deleted": &ddbTypes.AttributeValueMemberS{Value: models.StatusDeleted},
		},
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		Limit: dynamoLimit,
		ExclusiveStartKey: func() map[string]ddbTypes.AttributeValue {
			if nextToken == "" {
				return nil
			}
			return map[string]ddbTypes.AttributeValue{
				"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
				"sk": &ddbTypes.AttributeValueMemberS{Value: nextToken},
			}
		}(),
	})
	if err != nil {
		return nil, err
	}
	objects := make([]models.Object, 0, len(out.Items))
	for _, item := range out.Items {
		var obj models.Object
		if err := attributevalue.UnmarshalMap(item, &obj); err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}
	var lastEvalKey *string
	if out.LastEvaluatedKey != nil {
		if sk, ok := out.LastEvaluatedKey["sk"].(*ddbTypes.AttributeValueMemberS); ok {
			lastEvalKey = &sk.Value
		}
	}
	return &ScanResult{
		Objects:   objects,
		NextToken: lastEvalKey,
		Count:     int(out.Count),
	}, nil
}

func (d *DynamoClient) SoftDeleteObjectAndIndexes(ctx context.Context, tenantId string, tableHash string, objectId string) (*string, error) {
	pk := models.GenerateObjectPK(tenantId, tableHash)
	sk := models.GenerateObjectSK(objectId)

	now := time.Now().UTC()
	ttl := now.Add(30 * 24 * time.Hour).Unix() // delete after 30 days

	out, err := d.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.ObjectsTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbTypes.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:    aws.String("SET #status = :deleted, updated_at = :now, #v = #v + :inc, #ttl = :ttl"),
		ConditionExpression: aws.String("attribute_exists(pk) AND #status <> :deleted"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
			"#v":      "version",
			"#ttl":    "ttl",
		},
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":deleted": &ddbTypes.AttributeValueMemberS{Value: models.StatusDeleted},
			":now":     &ddbTypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":inc":     &ddbTypes.AttributeValueMemberN{Value: "1"},
			":ttl":     &ddbTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", ttl)},
		},
		ReturnValues: "ALL_NEW",
	})

	if err != nil {
		var condErr *ddbTypes.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil, ErrNotFoundOrDeleted
		}
		return nil, err
	}

	var obj models.Object
	err = attributevalue.UnmarshalMap(out.Attributes, &obj)
	if err != nil {
		return nil, err
	}

	go d.deleteIndexesByObject(ctx, tenantId, tableHash, objectId)

	if obj.S3Key != "" {
		return &obj.S3Key, nil
	}
	return nil, nil
}

func (d *DynamoClient) deleteIndexesByObject(ctx context.Context, tenantId string, tableHash string, objectId string) error {
	indexes, err := d.GetIndexesByObjectID(ctx, tenantId, tableHash, objectId)
	if err != nil {
		return err
	}
	idxs := make([]models.Index, 0, len(indexes))
	for _, idx := range indexes {
		idxs = append(idxs, idx)
	}

	for i := 0; i < len(idxs); i += 25 {
		end := min(i+25, len(idxs))
		writeRequests := make([]ddbTypes.WriteRequest, 0, end-i)
		for _, idx := range idxs[i:end] {
			writeRequests = append(writeRequests, ddbTypes.WriteRequest{
				DeleteRequest: &ddbTypes.DeleteRequest{
					Key: map[string]ddbTypes.AttributeValue{
						"pk": &ddbTypes.AttributeValueMemberS{Value: idx.PK},
						"sk": &ddbTypes.AttributeValueMemberB{Value: idx.SK},
					},
				},
			})
		}
		_, err := d.Client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbTypes.WriteRequest{
				d.IndexesTable: writeRequests,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DynamoClient) UndeleteObject(ctx context.Context, tenantId string, tableHash string, objectId string) (*string, error) {
	pk := models.GenerateObjectPK(tenantId, tableHash)
	sk := models.GenerateObjectSK(objectId)

	out, err := d.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.ObjectsTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbTypes.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:    aws.String("SET #status = :ready, updated_at = :now, #v = #v + :inc REMOVE #ttl"),
		ConditionExpression: aws.String("attribute_exists(pk) AND #status = :deleted"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
			"#v":      "version",
			"#ttl":    "ttl",
		},
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":deleted": &ddbTypes.AttributeValueMemberS{Value: models.StatusDeleted},
			":ready":   &ddbTypes.AttributeValueMemberS{Value: models.StatusReady},
			":now":     &ddbTypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			":inc":     &ddbTypes.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: "ALL_NEW",
	})
	if err != nil {
		var condErr *ddbTypes.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil, ErrNotFoundOrDeleted
		}
		return nil, err
	}
	var obj models.Object
	err = attributevalue.UnmarshalMap(out.Attributes, &obj)
	if err != nil {
		return nil, err
	}
	if obj.S3Key != "" {
		return &obj.S3Key, nil
	}
	return nil, nil
}

func (d *DynamoClient) QueryIndexes(ctx context.Context, tenantId string, tableHash string, index objects.Index, betweenRange [2][]byte, rangeOp objects.QueryOperator, limit int, nextToken string, scanForward bool) (*QueryResult, error) {
	var dynamoLimit *int32
	if limit > 0 {
		l := int32(limit)
		dynamoLimit = &l
	}
	var decodedNextToken []byte
	if rangeOp == objects.RangeBetween {
		if (betweenRange[0] == nil || betweenRange[1] == nil) || !index.IsHashOnly() {
			return nil, fmt.Errorf("invalid range query: for RangeBetween operator, index must be hash-only and both betweenRange tokens must be non-nil")
		}
	}
	if nextToken != "" {
		var err error
		decodedNextToken, err = base64.StdEncoding.DecodeString(nextToken)
		if err != nil {
			return nil, fmt.Errorf("invalid nextToken format: %w", err)
		}
		// validate token length based on index type
		if index.Name.RangeField != nil && len(decodedNextToken) != models.IndexTokenHashAndRangeLength {
			return nil, fmt.Errorf("invalid nextToken length for range index: expected %d, got %d", models.IndexTokenHashAndRangeLength, len(decodedNextToken))
		}
		if index.Name.RangeField == nil && len(decodedNextToken) != models.IndexTokenHashLength {
			return nil, fmt.Errorf("invalid nextToken length for hash-only index: expected %d, got %d", models.IndexTokenHashLength, len(decodedNextToken))
		}
	}

	var out *dynamodb.QueryOutput
	var err error

	// Query index table to get matching object IDs
	if index.Name.RangeField == nil {
		if !index.IsHashOnly() || betweenRange[0] != nil || betweenRange[1] != nil {
			return nil, fmt.Errorf("invalid index: for hash-only index, token must be non-nil and betweenRange must be nil")
		}
		// hash-only index, query by pk + token prefix
		pk := models.GenerateIndexPK(tenantId, tableHash, index.GetIndexName())
		out, err = d.Client.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(d.IndexesTable),
			KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
			ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
				":pk":     &ddbTypes.AttributeValueMemberS{Value: pk},
				":prefix": &ddbTypes.AttributeValueMemberB{Value: index.TokenHash},
			},
			Limit:            dynamoLimit,
			ScanIndexForward: aws.Bool(scanForward),
			ExclusiveStartKey: func() map[string]ddbTypes.AttributeValue {
				if decodedNextToken == nil {
					return nil
				}
				return map[string]ddbTypes.AttributeValue{
					"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
					"sk": &ddbTypes.AttributeValueMemberB{Value: decodedNextToken},
				}
			}(),
		})
		if err != nil {
			return nil, err
		}
	} else {
		// hash+range index, handle different range queries by translating to DynamoDB syntax
		if rangeOp == objects.RangeBetween {
			if !index.IsHashOnly() {
				return nil, fmt.Errorf("invalid index: for RangeBetween operator, index must be hash-only and betweenRange must be provided")
			}
			if len(betweenRange[0]) != models.OPERangeValueLength || len(betweenRange[1]) != models.OPERangeValueLength {
				return nil, fmt.Errorf("invalid betweenRange token length for range index: expected %d, got %d and %d", models.OPERangeValueLength, len(betweenRange[0]), len(betweenRange[1]))
			}
		}
		pk := models.GenerateIndexPK(tenantId, tableHash, index.GetIndexName())

		var keyCondExp string
		type AttributeValueMap = map[string]ddbTypes.AttributeValue
		var exprAttrValues AttributeValueMap = map[string]ddbTypes.AttributeValue{
			":pk": &ddbTypes.AttributeValueMemberS{Value: pk},
		}

		lower := make([]byte, models.DETHashValueLength, models.IndexTokenHashAndRangeLength)
		copy(lower, index.TokenHash[:models.DETHashValueLength])
		upper := make([]byte, models.DETHashValueLength, models.IndexTokenHashAndRangeLength)
		copy(upper, index.TokenHash[:models.DETHashValueLength])

		rangeVal := make([]byte, models.OPERangeValueLength)
		if rangeOp != objects.RangeBetween && rangeOp != objects.QueryEq {
			copy(rangeVal, index.TokenRange[:models.OPERangeValueLength])
		}

		switch rangeOp {
		case objects.QueryEq:
			keyCondExp = "pk = :pk AND begins_with(sk, :token)"
			exprAttrValues[":token"] = &ddbTypes.AttributeValueMemberB{Value: index.GetIndexToken()}
		case objects.RangeGt:
			keyCondExp = "pk = :pk AND sk BETWEEN :lower AND :upper"
			// lower: hash | rangeVal | max object ID (to exclude value with same rangeVal)
			// upper: hash | max range value | max object ID
			lower = append(lower, rangeVal...)
			lower = append(lower, models.MaxObjectID...)
			upper = append(upper, models.MaxRangeValue...)
			upper = append(upper, models.MaxObjectID...)
			exprAttrValues[":lower"] = &ddbTypes.AttributeValueMemberB{Value: lower}
			exprAttrValues[":upper"] = &ddbTypes.AttributeValueMemberB{Value: upper}
		case objects.RangeGte:
			keyCondExp = "pk = :pk AND sk BETWEEN :lower AND :upper"
			// lower: hash | rangeVal | min object ID (to include value with same rangeVal)
			// upper: hash | max range value | max object ID
			lower = append(lower, rangeVal...)
			lower = append(lower, models.MinObjectID...)
			upper = append(upper, models.MaxRangeValue...)
			upper = append(upper, models.MaxObjectID...)
			exprAttrValues[":lower"] = &ddbTypes.AttributeValueMemberB{Value: lower}
			exprAttrValues[":upper"] = &ddbTypes.AttributeValueMemberB{Value: upper}
		case objects.RangeLt:
			keyCondExp = "pk = :pk AND sk BETWEEN :lower AND :upper"
			// lower: hash | min range value | min object ID
			// upper: hash | rangeVal | min object ID (to exclude value with same rangeVal)
			lower = append(lower, models.MinRangeValue...)
			lower = append(lower, models.MinObjectID...)
			upper = append(upper, rangeVal...)
			upper = append(upper, models.MinObjectID...)
			exprAttrValues[":lower"] = &ddbTypes.AttributeValueMemberB{Value: lower}
			exprAttrValues[":upper"] = &ddbTypes.AttributeValueMemberB{Value: upper}
		case objects.RangeLte:
			keyCondExp = "pk = :pk AND sk BETWEEN :lower AND :upper"
			// lower: hash | min range value | min object ID
			// upper: hash | rangeVal | max object ID (to include value with same rangeVal)
			lower = append(lower, models.MinRangeValue...)
			lower = append(lower, models.MinObjectID...)
			upper = append(upper, rangeVal...)
			upper = append(upper, models.MaxObjectID...)
			exprAttrValues[":lower"] = &ddbTypes.AttributeValueMemberB{Value: lower}
			exprAttrValues[":upper"] = &ddbTypes.AttributeValueMemberB{Value: upper}
		case objects.RangeBetween:
			keyCondExp = "pk = :pk AND sk BETWEEN :lower AND :upper"
			// lower: hash | betweenRange[0] | min object ID
			// upper: hash | betweenRange[1] | max object ID
			lower = append(lower, betweenRange[0][:models.OPERangeValueLength]...)
			lower = append(lower, models.MinObjectID...)
			upper = append(upper, betweenRange[1][:models.OPERangeValueLength]...)
			upper = append(upper, models.MaxObjectID...)
			exprAttrValues[":lower"] = &ddbTypes.AttributeValueMemberB{Value: lower}
			exprAttrValues[":upper"] = &ddbTypes.AttributeValueMemberB{Value: upper}
		default:
			return nil, fmt.Errorf("unsupported range operator: %v", rangeOp)
		}
		out, err = d.Client.Query(ctx, &dynamodb.QueryInput{
			TableName:                 aws.String(d.IndexesTable),
			KeyConditionExpression:    aws.String(keyCondExp),
			ExpressionAttributeValues: exprAttrValues,
			Limit:                     dynamoLimit,
			ScanIndexForward:          aws.Bool(scanForward),
			ExclusiveStartKey: func() map[string]ddbTypes.AttributeValue {
				if decodedNextToken == nil {
					return nil
				}
				return map[string]ddbTypes.AttributeValue{
					"pk": &ddbTypes.AttributeValueMemberS{Value: pk},
					"sk": &ddbTypes.AttributeValueMemberB{Value: decodedNextToken},
				}
			}(),
		})
		if err != nil {
			return nil, err
		}
	}

	orderedIDs := make([]string, 0, len(out.Items))
	for _, item := range out.Items {
		var idx models.Index
		if err := attributevalue.UnmarshalMap(item, &idx); err != nil {
			return nil, err
		}
		orderedIDs = append(orderedIDs, idx.GetObjectID())
	}

	type batchResult struct {
		items []map[string]ddbTypes.AttributeValue
		err   error
	}
	batches := make([][]string, 0)
	for i := 0; i < len(orderedIDs); i += 100 {
		batches = append(batches, orderedIDs[i:min(i+100, len(orderedIDs))])
	}
	results := make([]batchResult, len(batches))
	var wg sync.WaitGroup
	for i, batch := range batches {
		wg.Add(1)
		go func(i int, batch []string) {
			defer wg.Done()
			keys := make([]map[string]ddbTypes.AttributeValue, len(batch))
			for j, id := range batch {
				keys[j] = map[string]ddbTypes.AttributeValue{
					"pk": &ddbTypes.AttributeValueMemberS{Value: models.GenerateObjectPK(tenantId, tableHash)},
					"sk": &ddbTypes.AttributeValueMemberS{Value: models.GenerateObjectSK(id)},
				}
			}

			var allItems []map[string]ddbTypes.AttributeValue
			remaining := map[string]ddbTypes.KeysAndAttributes{
				d.ObjectsTable: {Keys: keys, ConsistentRead: aws.Bool(true)},
			}

			retryDelay := 50 * time.Millisecond
			for len(remaining) > 0 {
				batchOut, err := d.Client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
					RequestItems: remaining,
				})
				if err != nil {
					results[i] = batchResult{nil, err}
					return
				}
				allItems = append(allItems, batchOut.Responses[d.ObjectsTable]...)
				remaining = batchOut.UnprocessedKeys

				if len(remaining) > 0 {
					select {
					case <-ctx.Done():
						results[i] = batchResult{nil, ctx.Err()}
						return
					case <-time.After(retryDelay):
						retryDelay = min(retryDelay*2, 1*time.Second)
					}
				}
			}

			results[i] = batchResult{items: allItems, err: nil}
		}(i, batch)
	}
	wg.Wait()

	objectsByID := make(map[string]models.Object, len(orderedIDs))
	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}
		for _, item := range result.items {
			var obj models.Object
			if err := attributevalue.UnmarshalMap(item, &obj); err != nil {
				return nil, err
			}
			objectsByID[obj.GetObjectID()] = obj
		}
	}

	objects := make([]models.Object, 0, len(out.Items))
	for _, id := range orderedIDs {
		if obj, ok := objectsByID[id]; ok {
			objects = append(objects, obj)
		}
		if dynamoLimit != nil && len(objects) == int(*dynamoLimit) {
			break
		}
	}

	var newNextToken *string
	if out.LastEvaluatedKey != nil {
		if sk, ok := out.LastEvaluatedKey["sk"].(*ddbTypes.AttributeValueMemberB); ok {
			s := base64.StdEncoding.EncodeToString(sk.Value)
			newNextToken = &s
		}
	}

	return &QueryResult{
		Objects:   objects,
		Count:     int(out.Count),
		NextToken: newNextToken,
	}, nil
}
