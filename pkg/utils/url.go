package utils

import "net/url"

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
