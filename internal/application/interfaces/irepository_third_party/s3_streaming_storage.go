package irepository_third_party

import (
	"context"
	"io"
)

type S3StreamingStorage interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
	BuildUrl(key string) string
}
