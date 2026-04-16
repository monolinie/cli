package home

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
	APIKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		http:    &http.Client{},
	}
}

type RegisterInput struct {
	Name              string `json:"name"`
	Subdomain         string `json:"subdomain"`
	GithubRepo        string `json:"githubRepo,omitempty"`
	DokployProjectID  string `json:"dokployProjectId"`
	DokployAppID      string `json:"dokployAppId,omitempty"`
	DokployPostgresID string `json:"dokployPostgresId,omitempty"`
}

type RegisterResult struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type DeregisterInput struct {
	DokployProjectID string `json:"dokployProjectId,omitempty"`
	Name             string `json:"name,omitempty"`
}

type DeregisterResult struct {
	OK      bool    `json:"ok"`
	Deleted *string `json:"deleted"`
}

type SyncResult struct {
	Created   []SyncEntry  `json:"created"`
	Updated   []SyncUpdate `json:"updated"`
	Orphaned  []SyncEntry  `json:"orphaned"`
	Unchanged int          `json:"unchanged"`
	Errors    []SyncError  `json:"errors"`
}

type SyncEntry struct {
	Name             string `json:"name"`
	ID               string `json:"id"`
	DokployProjectID string `json:"dokployProjectId"`
}

type SyncUpdate struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Field string `json:"field"`
}

type SyncError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

func (c *Client) RegisterProject(input RegisterInput) (*RegisterResult, error) {
	var result RegisterResult
	if err := c.doRequest("POST", "/api/cli/projects", input, &result); err != nil {
		return nil, fmt.Errorf("register project: %w", err)
	}
	return &result, nil
}

func (c *Client) DeregisterProject(input DeregisterInput) (*DeregisterResult, error) {
	var result DeregisterResult
	if err := c.doRequest("DELETE", "/api/cli/projects", input, &result); err != nil {
		return nil, fmt.Errorf("deregister project: %w", err)
	}
	return &result, nil
}

func (c *Client) Sync() (*SyncResult, error) {
	var result SyncResult
	if err := c.doRequest("POST", "/api/cli/sync", nil, &result); err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}
	return &result, nil
}

func (c *Client) doRequest(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
