package dokploy

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

type Project struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
}

type CreateProjectResponse struct {
	Project     Project     `json:"project"`
	Environment Environment `json:"environment"`
}

type Application struct {
	ApplicationID string `json:"applicationId"`
	AppName       string `json:"appName"`
}

type Postgres struct {
	PostgresID string `json:"postgresId"`
}

type Environment struct {
	EnvironmentID string `json:"environmentId"`
	Name          string `json:"name"`
}

type SSHKey struct {
	SSHKeyID  string `json:"sshKeyId"`
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type Domain struct {
	DomainID string `json:"domainId"`
}

func (c *Client) CreateProject(name, description string) (*CreateProjectResponse, error) {
	body := map[string]string{
		"name":        name,
		"description": description,
	}
	var result CreateProjectResponse
	if err := c.post("/api/project.create", body, &result); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &result, nil
}

func (c *Client) GetEnvironments(projectID string) ([]Environment, error) {
	var result []Environment
	if err := c.get("/api/environment.byProjectId?projectId="+projectID, &result); err != nil {
		return nil, fmt.Errorf("get environments: %w", err)
	}
	return result, nil
}

func (c *Client) CreateApplication(environmentID, name, description string) (*Application, error) {
	body := map[string]string{
		"environmentId": environmentID,
		"name":          name,
		"description":   description,
	}
	var result Application
	if err := c.post("/api/application.create", body, &result); err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}
	return &result, nil
}

func (c *Client) SaveGitProvider(appID, gitURL, branch, buildPath, sshKeyID string) error {
	body := map[string]any{
		"applicationId":      appID,
		"customGitUrl":       gitURL,
		"customGitBranch":    branch,
		"customGitBuildPath": buildPath,
		"customGitSSHKeyId":  sshKeyID,
		"watchPaths":         nil,
	}
	return c.post("/api/application.saveGitProvider", body, nil)
}

func (c *Client) GetSSHKeys() ([]SSHKey, error) {
	var result []SSHKey
	if err := c.get("/api/sshKey.all", &result); err != nil {
		return nil, fmt.Errorf("get ssh keys: %w", err)
	}
	return result, nil
}

func (c *Client) CreatePostgres(environmentID, name, dbName, dbUser, dbPassword string) (*Postgres, error) {
	body := map[string]string{
		"environmentId":    environmentID,
		"name":             name,
		"databaseName":     dbName,
		"databaseUser":     dbUser,
		"databasePassword": dbPassword,
	}
	var result Postgres
	if err := c.post("/api/postgres.create", body, &result); err != nil {
		return nil, fmt.Errorf("create postgres: %w", err)
	}
	return &result, nil
}

func (c *Client) DeployPostgres(postgresID string) error {
	body := map[string]string{
		"postgresId": postgresID,
	}
	return c.post("/api/postgres.deploy", body, nil)
}

func (c *Client) CreateDomain(appID, host string, port int, https bool, certificateType string) (*Domain, error) {
	body := map[string]any{
		"applicationId":   appID,
		"host":            host,
		"port":            port,
		"https":           https,
		"certificateType": certificateType,
	}
	var result Domain
	if err := c.post("/api/domain.create", body, &result); err != nil {
		return nil, fmt.Errorf("create domain: %w", err)
	}
	return &result, nil
}

func (c *Client) SaveEnvironment(appID, envVars string) error {
	body := map[string]any{
		"applicationId": appID,
		"env":           envVars,
		"buildArgs":     "",
		"buildSecrets":  "",
		"createEnvFile": false,
	}
	return c.post("/api/application.saveEnvironment", body, nil)
}

func (c *Client) DeployApplication(appID string) error {
	body := map[string]string{
		"applicationId": appID,
	}
	return c.post("/api/application.deploy", body, nil)
}

func (c *Client) SaveBuildType(appID, buildType, dockerfile string) error {
	body := map[string]any{
		"applicationId":    appID,
		"buildType":        buildType,
		"dockerfile":       dockerfile,
		"dockerContextPath": ".",
		"dockerBuildStage":  "",
		"herokuVersion":     "",
		"railpackVersion":   "",
	}
	return c.post("/api/application.saveBuildType", body, nil)
}

func (c *Client) get(path string, result any) error {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.APIKey)

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

func (c *Client) post(path string, body any, result any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)

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
