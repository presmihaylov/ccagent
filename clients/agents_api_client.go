package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AttachmentResponse represents the API response for fetching an attachment
type AttachmentResponse struct {
	ID   string `json:"id"`
	Data string `json:"data"` // Base64-encoded content
}

// AgentsApiClient handles API requests to the Claude Control agents API
type AgentsApiClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAgentsApiClient creates a new agents API client
func NewAgentsApiClient(apiKey, baseURL string) *AgentsApiClient {
	return &AgentsApiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAttachment fetches an attachment by ID from the Claude Control API
func (c *AgentsApiClient) FetchAttachment(attachmentID string) (*AttachmentResponse, error) {
	url := fmt.Sprintf("%s/api/agents/attachments/%s", c.baseURL, attachmentID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Bearer token authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var attachmentResp AttachmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&attachmentResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &attachmentResp, nil
}
