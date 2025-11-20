package third_party_repository

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/infrastructure/persistence"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Storage struct {
	config     *config.AppConfig
	client     *s3.Client
	bucketName string
	region     string
}

func NewS3Storage(config *config.AppConfig, bucket *persistence.S3Bucket) irepository_third_party.S3Storage {
	return &s3Storage{
		config:     config,
		client:     bucket.Client,
		bucketName: bucket.BucketName,
		region:     bucket.Region,
	}
}

func (r *s3Storage) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
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

// Get retrieves an object from S3 and returns a reader and the content length
func (r *s3Storage) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object %s: %w", key, err)
	}

	// Get content length from metadata
	contentLength := int64(0)
	if output.ContentLength != nil {
		contentLength = *output.ContentLength
	}

	return output.Body, contentLength, nil
}

func (r *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

// BuildUrl constructs the S3 URL for the given key.
//
//	@key:	should have the format "<user_id>/<timestamp>_filename.ext"
func (r *s3Storage) BuildUrl(key string) string {
	if r.config.S3Bucket.Endpoint != "" && strings.Contains(r.config.S3Bucket.Endpoint, "localhost") {
		return fmt.Sprintf("%s/%s/%s", r.config.S3Bucket.Endpoint, r.bucketName, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", r.bucketName, r.region, key)
}
