package storage

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client    *s3.Client
	Bucket    string
	presigner *s3.PresignClient
}

func NewS3Client(ctx context.Context, bucket string, cfg aws.Config, useLocalstack bool, localstackUrl string) *S3Client {
	// add localstack endpoint resolver if needed
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		if useLocalstack {
			o.BaseEndpoint = aws.String(localstackUrl)
		}
	})
	return &S3Client{
		client:    client,
		Bucket:    bucket,
		presigner: s3.NewPresignClient(client),
	}
}

func (s *S3Client) TestConnnectivity(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.Bucket),
	})
	return err
}

/*
PresignPutObjectUrl generates a presigned URL for uploading an object to S3.
*/
func (s *S3Client) PresignPutObjectUrl(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	request, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/json"),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = lifetime
	})
	if err != nil {
		return "", err
	}
	if request.URL == "" {
		return "", errors.New("failed to generate presigned URL")
	}
	return request.URL, nil
}
