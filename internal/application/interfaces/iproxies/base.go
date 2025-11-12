// Package iproxies contains the interfaces for the proxies in the application.
package iproxies

import (
	"context"
	"net/http"
)

type BaseProxy interface {
	Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error)
	Post(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error)
	Put(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error)
	Patch(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error)
	Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error)
	SetBaseURL(baseURL string)
	HandleNon2xxHTTPResponse(resp *http.Response) error
}
