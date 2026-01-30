package processor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dynamoClient *dynamodb.Client
var tableName string

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
	tableName = os.Getenv("DYNAMO_OBJECTS_TABLE")
}

func UpdateStatus(ctx context.Context, bucket string, key string) error {
	var tenantId, tableHash, objectId string
	// extract tenantId and objectId from key
	err := extractFromKey(key, &tenantId, &tableHash, &objectId)
	if err != nil {
		return err
	}
	_, err = dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &tableName,
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "TENANT#" + tenantId + "#TABLE#" + tableHash},
			"sk": &ddbTypes.AttributeValueMemberS{Value: "OBJ#" + objectId},
		},
		UpdateExpression: aws.String("SET #status = :ready REMOVE #ttl"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
			"#ttl":    "TTL",
		},
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":ready": &ddbTypes.AttributeValueMemberS{Value: models.StatusReady},
		},
	})
	return err
}

func extractFromKey(key string, tenantId *string, tableHash *string, objectId *string) error {
	// key format: tenant-<tenant_id>/<table_hash>/<object_id>/v{n}
	parts := strings.Split(key, "/")
	if len(parts) != 4 || !strings.HasPrefix(parts[0], "tenant-") || !strings.HasPrefix(parts[3], "v") {
		return fmt.Errorf("invalid S3 key format: %s", key)
	}

	*tenantId = strings.TrimPrefix(parts[0], "tenant-")
	*tableHash = parts[1]
	*objectId = parts[2]
	return nil
}
