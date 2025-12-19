package main

import (
	"context"
	"log"

	processor "github.com/QodeSrl/gardbase/apps/lambdas/upload-processor/internal"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

/*
This is the main entry point for the upload processor Lambda function.
It listens for an S3 create event, which indicates that a new object has been uploaded to S3 through the given presigned url.
For each record in the S3 event, it calls the UpdateStatus function from the processor package to update the object's status in DynamoDB from "pending" to "ready" and remove the ttl attribute.
*/
func handler(ctx context.Context, s3Event events.S3Event) error {
	for _, record := range s3Event.Records {
		err := processor.UpdateStatus(ctx, record.S3.Bucket.Name, record.S3.Object.Key)
		if err != nil {
			log.Printf("failed to process %s: %v", record.S3.Object.Key, err)
			return err
		}
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
