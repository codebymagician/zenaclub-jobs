package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Ported from ringly-jobs/crm-jobs/utils/api-client.go, trimmed to the one
// method subscription-due-charges.go actually calls (Post) -- Get/Patch can
// come back the same way crm-jobs has them if a future job needs them.
type APIClient struct {
	BaseURL string
	Client  *http.Client
	Headers map[string]string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (api *APIClient) Post(endpoint string, payload interface{}) (int, []byte, error) {
	url := fmt.Sprintf("%s%s", api.BaseURL, endpoint)

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range api.Headers {
		req.Header.Set(key, value)
	}

	resp, err := api.Client.Do(req)
	if err != nil {
		// %w, not %v: callers need errors.As to reach the underlying
		// net.Error to distinguish a timeout from a refused connection.
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, respBody, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return resp.StatusCode, respBody, nil
}
