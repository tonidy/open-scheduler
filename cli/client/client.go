package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultBaseURL = "http://localhost:8080/api/v1"
	TokenFile      = ".osctl_token"
)

type Client struct {
	BaseURL string
	Token   string
	client  *http.Client
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
	Username  string `json:"username"`
}

func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) LoadToken() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tokenPath := filepath.Join(homeDir, TokenFile)
	
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Token file doesn't exist, will need to login
		}
		return err
	}
	
	c.Token = string(bytes.TrimSpace(data))
	return nil
}

func (c *Client) SaveToken(token string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tokenPath := filepath.Join(homeDir, TokenFile)
	
	c.Token = token
	return os.WriteFile(tokenPath, []byte(token), 0600)
}

func (c *Client) Login(username, password string) error {
	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}
	
	url := fmt.Sprintf("%s/auth/login", c.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s (status: %d)", string(body), resp.StatusCode)
	}
	
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	return c.SaveToken(loginResp.Token)
}

func (c *Client) ensureAuthenticated() error {
	if c.Token == "" {
		// Try to load from file
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("not authenticated. Please login first")
		}
		if c.Token == "" {
			return fmt.Errorf("not authenticated. Please login first")
		}
	}
	return nil
}

func (c *Client) DoRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	return resp, nil
}

func (c *Client) Get(endpoint string) (map[string]interface{}, error) {
	resp, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		var errorResp map[string]string
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if msg, ok := errorResp["error"]; ok {
				return nil, fmt.Errorf("API error: %s (status: %d)", msg, resp.StatusCode)
			}
		}
		return nil, fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result, nil
}

func (c *Client) Post(endpoint string, body interface{}) (map[string]interface{}, error) {
	resp, err := c.DoRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		var errorResp map[string]string
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if msg, ok := errorResp["error"]; ok {
				return nil, fmt.Errorf("API error: %s (status: %d)", msg, resp.StatusCode)
			}
		}
		return nil, fmt.Errorf("API error: %s (status: %d)", string(respBody), resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result, nil
}

