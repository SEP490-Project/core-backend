package irepository_third_party

import (
	"context"
	"io"
)

type S3Repository interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
	BuildUrl(key string) string
}
