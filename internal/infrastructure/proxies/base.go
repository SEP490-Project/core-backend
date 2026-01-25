package proxies

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

type BaseProxy struct {
	httpClient *http.Client
	baseURL    string
	config     *config.AppConfig
}

func NewBaseProxy(httpClient *http.Client, baseURL string, config *config.AppConfig) *BaseProxy {
	return &BaseProxy{
		httpClient: httpClient,
		baseURL:    baseURL,
		config:     config,
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
	endpoint, err := p.urlResolution(path)
	if err != nil {
		zap.L().Debug("Failed to construct URL", zap.String("baseURL", p.baseURL), zap.String("path", path), zap.Error(err))
		return nil, fmt.Errorf("invalid URL path: %w", err)
	}

	// Marshal body if present
	var bodyReader io.Reader
	var contentType string
	if body != nil {
		for k, v := range headers {
			if strings.EqualFold(k, "content-type") {
				contentType = v
				break
			}
		}

		switch v := body.(type) {
		case io.Reader:
			bodyReader = v
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		case []byte:
			bodyReader = bytes.NewReader(v)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		case url.Values:
			bodyReader = strings.NewReader(v.Encode())
			if contentType == "" {
				contentType = "application/x-www-form-urlencoded"
			}
		case utils.MultipartFormBuilder:
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			// Call the builder function to populate the form
			if err = v(writer); err != nil {
				return nil, fmt.Errorf("failed to build multipart form: %w", err)
			}

			if err = writer.Close(); err != nil {
				return nil, fmt.Errorf("failed to close multipart writer: %w", err)
			}

			bodyReader = &buf
			contentType = writer.FormDataContentType()
			zap.L().Debug("Raw multipart request details",
				zap.String("Content-Type-Header", contentType),
				zap.String("Request-Body", buf.String()),
			)
		default:
			var data []byte
			data, err = json.Marshal(v)
			if err != nil {
				zap.L().Debug("Failed to marshal request body", zap.Error(err))
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(data)
			if contentType == "" {
				contentType = "application/json"
			}
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Default headers
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if body != nil {
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
	p.handleDebugResponseLog(resp)
	return resp, nil
}

// region: ======== BaseProxy Common HTTP Methods ========

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

// endregion

// region: ======== Generic HTTP Helper Functions ========

func GetGeneric[T any](p iproxies.BaseProxy, ctx context.Context, path string, headers map[string]string, response *T) error {
	resp, err := p.Get(ctx, path, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := p.HandleNon2xxHTTPResponse(resp); err != nil {
		return err
	}

	// Parse response
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	zap.L().Debug("GET request successful", zap.String("path", path), zap.Any("response", response))

	return nil
}

// PostGeneric makes a POST request and decodes the response into the provided type
func PostGeneric[T any](p iproxies.BaseProxy, ctx context.Context, path string, headers map[string]string, body any, response *T) error {
	resp, err := p.Post(ctx, path, headers, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := p.HandleNon2xxHTTPResponse(resp); err != nil {
		return err
	}

	// Parse response
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	zap.L().Debug("POST request successful", zap.String("path", path), zap.Any("response", response))

	return nil
}

// endregion

func (p *BaseProxy) urlResolution(path string) (string, error) {
	base, err := url.Parse(p.baseURL)
	if err != nil {
		zap.L().Error("Failed to parse base URL", zap.String("baseURL", p.baseURL), zap.Error(err))
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Parse the relative path and query string.
	rel, err := url.Parse(path)
	if err != nil {
		zap.L().Debug("Failed to parse relative path", zap.String("path", path), zap.Error(err))
		return "", fmt.Errorf("invalid relative path: %w", err)
	}

	// ResolveReference correctly combines the base URL with the relative path and query.
	return base.ResolveReference(rel).String(), nil
}

func (p *BaseProxy) HandleNon2xxHTTPResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
}

func (p *BaseProxy) handleDebugResponseLog(resp *http.Response) {
	if p.config.IsDevelopmentDebugging() {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			zap.L().Debug("Failed to read body for debug logging", zap.Error(err))
			return
		}
		var data map[string]any
		if err := json.Unmarshal(bodyBytes, &data); err == nil {
			zap.L().Debug("response body decoded", zap.Any("response_body", data))
		} else {
			zap.L().Debug("Failed to decode response body", zap.Error(err))
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
}
