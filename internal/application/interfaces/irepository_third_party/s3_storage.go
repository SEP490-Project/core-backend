package irepository_third_party

import (
	"context"
	"io"
)

type S3Storage interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error) // Returns reader, file size, error
	Delete(ctx context.Context, key string) error
	BuildUrl(key string) string
}
