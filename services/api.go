package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseSize = 10 << 20

type APIClient struct {
	endpoint  string
	parameter string
	client    *http.Client
}

func NewAPIClient(endpoint, parameter string, timeout time.Duration) *APIClient {
	return &APIClient{
		endpoint:  strings.TrimSpace(endpoint),
		parameter: strings.TrimSpace(parameter),
		client:    &http.Client{Timeout: timeout},
	}
}

func (c *APIClient) Fetch(ctx context.Context, value string) ([]byte, error) {
	if c.endpoint == "" {
		return mockResponse(value), nil
	}

	endpoint, err := url.Parse(c.endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid API_URL %q", c.endpoint)
	}
	query := endpoint.Query()
	query.Set(c.parameter, value)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read API response: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("API response exceeds %d bytes", maxResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func mockResponse(value string) []byte {
	payload := []map[string]any{
		{
			"id":       1,
			"query":    value,
			"title":    "Getting started",
			"category": "guide",
		},
		{
			"id":       2,
			"query":    value,
			"title":    "Mock JSON response",
			"category": "example",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return data
}
