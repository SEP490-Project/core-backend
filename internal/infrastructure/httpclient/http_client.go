package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type GeneralAPIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// DoRequestSingle handles responses with a single object in "data"
func DoRequestSingle[T any](ctx context.Context, client *http.Client, token string, method, url string, body any) (T, error) {
	var zero T
	var buf io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("marshal body: %w", err)
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return zero, fmt.Errorf("new request: %w", err)
	}
	req.Header.Add("Token", token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Warn("Non-200 from external service",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(b)),
		)
	}

	var result GeneralAPIResponse[T]
	if err := json.Unmarshal(b, &result); err != nil {
		return zero, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}

// DoRequestList handles responses with "data" being an array []T
func DoRequestList[T any](ctx context.Context, client *http.Client, token string, method, url string, body any) ([]T, error) {
	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Add("Token", token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Warn("Non-200 from external service",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(b)),
		)
	}

	var result GeneralAPIResponse[[]T]
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return result.Data, nil
}
