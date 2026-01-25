package irepository_third_party

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"io"
)

type S3StreamingStorage interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	BatchPut(ctx context.Context, items []dtos.BatchUploadItem) error
	Delete(ctx context.Context, key string) error
	BuildUrl(key string) string
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
}
