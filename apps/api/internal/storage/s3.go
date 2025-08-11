package storage

import (
	"context"
	"errors"
	"time"

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

func (s *S3Client) PresignPutUrl(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "", errors.New("not implemented")
}