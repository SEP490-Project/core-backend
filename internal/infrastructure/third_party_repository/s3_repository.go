package third_party_repository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/infrastructure/persistence"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
)

type s3Repository struct {
	client     *s3.Client
	bucketName string
	region     string
}

func NewS3Repository(bucket *persistence.S3Bucket) irepository_third_party.S3Repository {
	return &s3Repository{
		client:     bucket.Client,
		bucketName: bucket.BucketName,
		region:     bucket.Region,
	}
}

func (r *s3Repository) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return fmt.Errorf("failed to put object %s: %w", key, err)
	}
	return nil
}

func (r *s3Repository) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

func (r *s3Repository) BuildUrl(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", r.bucketName, r.region, key)
}
