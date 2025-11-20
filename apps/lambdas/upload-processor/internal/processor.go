package processor

import (
	"context"
	"fmt"
	"os"

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
	var tenantId, objectId string
	// extract tenantId and objectId from key
	fmt.Sscanf(key, "tenant-%s/objects/%s/v%d", &tenantId, &objectId)
	_, err := dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &tableName,
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: "TENANT#" + tenantId},
			"sk": &ddbTypes.AttributeValueMemberS{Value: "OBJ#" + key},
		},
		UpdateExpression: aws.String("SET #status = :ready REMOVE TTL"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":ready": &ddbTypes.AttributeValueMemberS{Value: models.StatusReady},
		},
	})
	return err
}
