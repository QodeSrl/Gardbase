package storage

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func (s *S3Client) PresignGetObjectUrl(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
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

func (s *S3Client) CheckObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFoundErr *s3Types.NotFound
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Client) TagForDeletion(ctx context.Context, key string) error {
	_, err := s.client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Tagging: &s3Types.Tagging{
			TagSet: []s3Types.Tag{
				{
					Key:   aws.String("status"),
					Value: aws.String("deleted"),
				},
				{
					Key:   aws.String("deleted_at"),
					Value: aws.String(time.Now().Format(time.RFC3339)),
				},
			},
		},
	})
	return err
}

func (s *S3Client) UntagForDeletion(ctx context.Context, key string) error {
	_, err := s.client.DeleteObjectTagging(ctx, &s3.DeleteObjectTaggingInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	return err
}
