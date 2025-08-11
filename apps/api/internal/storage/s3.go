package storage

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(client *s3.Client, bucket string) *S3Client {
	return &S3Client{
		client: client,
		bucket: bucket,
	}
}
