package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// AddQueryParam appends a single query parameter to the given URL.
func AddQueryParam[T comparable](url, key string, value T) (string, error) {
	return AddQueryParams(url, map[string]T{key: value})
}

// AddQueryParams appends multiple query parameters to the given URL.
func AddQueryParams[T comparable](rawURL string, params map[string]T) (string, error) {
	// Parse the URL to ensure it's valid
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Extract and modify query params
	q := parsedURL.Query()
	for key, value := range params {
		q.Set(key, ToString(value))
	}

	parsedURL.RawQuery = q.Encode()
	return parsedURL.String(), nil
}

type MultipartFormBuilder func(*multipart.Writer) error

// NewMultipartForm creates a MultipartFormBuilder that adds only fields to the multipart form.
func NewMultipartForm(fields map[string]string) MultipartFormBuilder {
	return func(writer *multipart.Writer) error {
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				return fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}
		return nil
	}
}

// NewMultipartFormWithFiles creates a MultipartFormBuilder that adds both fields and files to the multipart form.
func NewMultipartFormWithFiles(fields map[string]string, files map[string]io.Reader) MultipartFormBuilder {
	return func(writer *multipart.Writer) error {
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				return fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}

		for key, file := range files {
			part, err := writer.CreateFormFile(key, key)
			if err != nil {
				return fmt.Errorf("failed to create form file %s: %w", key, err)
			}
			if _, err := io.Copy(part, file); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", key, err)
			}
		}
		return nil
	}
}

// EncodeIndividualPathSegments encodes each segment of the URL path using NFC normalization.
func EncodeIndividualPathSegments(urlStr string) (string, error) {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	pathSegments := strings.Split(strings.Trim(url.Path, "/"), "/")
	for i, segment := range pathSegments {
		pathSegments[i] = norm.NFC.String(segment)
	}
	url.Path = "/" + strings.Join(pathSegments, "/")

	return url.String(), nil
}
