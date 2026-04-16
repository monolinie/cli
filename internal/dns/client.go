package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Hetzner Cloud API (replaces the deprecated dns.hetzner.com API)
const baseURL = "https://api.hetzner.cloud/v1"

type Client struct {
	Token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
		http:  &http.Client{},
	}
}

type Zone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type zonesResponse struct {
	Zones []Zone `json:"zones"`
}

func (c *Client) GetZoneByName(name string) (*Zone, error) {
	req, err := http.NewRequest("GET", baseURL+"/zones?name="+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get zones: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result zonesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Zones) == 0 {
		return nil, fmt.Errorf("zone not found: %s", name)
	}

	return &result.Zones[0], nil
}

type recordValue struct {
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
}

type createRecordRequest struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	TTL     *int          `json:"ttl"`
	Records []recordValue `json:"records"`
}

// DeleteRecord deletes all records matching the given name and type in a zone.
func (c *Client) DeleteRecord(zoneID int, recordType, name string) error {
	url := fmt.Sprintf("%s/zones/%d/rrsets/%s/%s", baseURL, zoneID, name, recordType)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) CreateRecord(zoneID int, recordType, name, value string, ttl int) error {
	payload := createRecordRequest{
		Name: name,
		Type: recordType,
		TTL:  &ttl,
		Records: []recordValue{
			{Value: value, Comment: "Created by monolinie CLI"},
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/zones/%d/rrsets", baseURL, zoneID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("create record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
