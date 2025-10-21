package third_party_repository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/infrastructure/persistence"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3StreamingStorage struct {
	client           *s3.Client
	bucketName       string
	region           string
	cloudfrontDomain string
}

func (s *s3StreamingStorage) List(ctx context.Context, prefix string) ([]string, error) {
	results := make([]string, 0)
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects with prefix %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				results = append(results, *obj.Key)
			}
		}
	}

	return results, nil
}

func (s *s3StreamingStorage) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	//Get content length
	file, ok := body.(*os.File)
	if !ok {
		return fmt.Errorf("body must be *os.File to set ContentLength")
	}
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(info.Size()),
		ACL:           types.ObjectCannedACLPrivate,
	})

	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to put object in stream bucket: %s", err.Error()))
		return fmt.Errorf("failed to put object %s: %w", key, err)
	}
	return nil
}

func (s *s3StreamingStorage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

func (s *s3StreamingStorage) BuildUrl(key string) string {
	if s.cloudfrontDomain != "" {
		return strings.TrimRight(s.cloudfrontDomain, "/") + "/" + strings.TrimLeft(key, "/")
	}
	// fallback URL
	return fmt.Sprintf("https://cdn.com/%s", s.bucketName, s.region, key)
}

func NewS3StreamingStorage(s3StreamBucket *persistence.S3StreamingBucket) irepository_third_party.S3StreamingStorage {
	return &s3StreamingStorage{
		client:           s3StreamBucket.Client,
		bucketName:       s3StreamBucket.BucketName,
		region:           s3StreamBucket.Region,
		cloudfrontDomain: s3StreamBucket.CloudfrontDomain,
	}
}
