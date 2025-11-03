package proxies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

type BaseProxy struct {
	httpClient *http.Client
	baseURL    string
}

func NewBaseProxy(httpClient *http.Client, baseURL string) *BaseProxy {
	return &BaseProxy{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// doRequest handles all outgoing HTTP requests.
func (p *BaseProxy) doRequest(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	body any,
) (*http.Response, error) {
	// Construct full URL
	endpoint, err := url.JoinPath(p.baseURL, path)
	if err != nil {
		zap.L().Debug("Failed to construct URL", zap.String("baseURL", p.baseURL), zap.String("path", path), zap.Error(err))
		return nil, fmt.Errorf("invalid URL path: %w", err)
	}

	// Marshal body if present
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case io.Reader:
			bodyReader = v
		default:
			var data []byte
			data, err = json.Marshal(v)
			if err != nil {
				zap.L().Debug("Failed to marshal request body", zap.Error(err))
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(data)
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Default headers
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Merge custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		zap.L().Debug("HTTP request failed", zap.Error(err), zap.String("method", method), zap.String("url", endpoint))
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	zap.L().Debug("HTTP request completed",
		zap.String("method", method),
		zap.String("url", endpoint),
		zap.Int("status_code", resp.StatusCode),
	)
	return resp, nil
}

// ---- Common HTTP methods ----

// Get make a GET request to the API with the specified path and headers
func (p *BaseProxy) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return p.doRequest(ctx, http.MethodGet, path, headers, nil)
}

// Post make a POST request to the API with the specified path, headers, and body
func (p *BaseProxy) Post(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error) {
	return p.doRequest(ctx, http.MethodPost, path, headers, body)
}

// Put make a PUT request to the API with the specified path, headers, and body
func (p *BaseProxy) Put(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error) {
	return p.doRequest(ctx, http.MethodPut, path, headers, body)
}

// Patch make a PATCH request to the API with the specified path, headers, and body
func (p *BaseProxy) Patch(ctx context.Context, path string, headers map[string]string, body any) (*http.Response, error) {
	return p.doRequest(ctx, http.MethodPatch, path, headers, body)
}

// Delete make a DELETE request to the API with the specified path and headers
func (p *BaseProxy) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return p.doRequest(ctx, http.MethodDelete, path, headers, nil)
}

// SetBaseURL updates the base URL for the proxy
func (p *BaseProxy) SetBaseURL(baseURL string) {
	p.baseURL = baseURL
}
