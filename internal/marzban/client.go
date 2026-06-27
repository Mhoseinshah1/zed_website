package marzban

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a Marzban API client with token caching.
type Client struct {
	BaseURL  string
	Username string
	Password string
	token    string
	tokenExp time.Time
	http     *http.Client
}

// New creates a new Marzban client.
func New(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) auth() error {
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return nil
	}
	form := url.Values{}
	form.Set("username", c.Username)
	form.Set("password", c.Password)
	resp, err := c.http.PostForm(c.BaseURL+"/api/admin/token", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("auth failed: HTTP %d", resp.StatusCode)
	}
	var r struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}
	c.token = r.AccessToken
	c.tokenExp = time.Now().Add(50 * time.Minute)
	return nil
}

func (c *Client) do(method, path string, body interface{}) ([]byte, int, error) {
	if err := c.auth(); err != nil {
		return nil, 0, err
	}
	var br io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		br = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, br)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

// CreateUserReq is the request body for creating a Marzban user.
type CreateUserReq struct {
	Username  string              `json:"username"`
	Proxies   map[string]any      `json:"proxies"`
	Inbounds  map[string][]string `json:"inbounds"`
	Expire    int64               `json:"expire"`    // unix timestamp, 0=unlimited
	DataLimit int64               `json:"data_limit"` // bytes, 0=unlimited
	Status    string              `json:"status"`    // "active"
	Note      string              `json:"note"`
}

// UserResponse is returned by Marzban for user operations.
type UserResponse struct {
	Username        string   `json:"username"`
	Status          string   `json:"status"`
	SubscriptionURL string   `json:"subscription_url"`
	Links           []string `json:"links"`
	Expire          int64    `json:"expire"`
	DataLimit       int64    `json:"data_limit"`
	UsedTraffic     int64    `json:"used_traffic"`
}

// TestConnection checks connectivity to the Marzban panel.
func (c *Client) TestConnection() error {
	_, status, err := c.do("GET", "/api/admin", nil)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("unexpected status %d", status)
	}
	return nil
}

// CreateUser creates a new user on the Marzban panel.
func (c *Client) CreateUser(req CreateUserReq) (*UserResponse, error) {
	b, status, err := c.do("POST", "/api/user", req)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("create user failed: HTTP %d: %s", status, string(b))
	}
	var u UserResponse
	json.Unmarshal(b, &u) //nolint:errcheck
	return &u, nil
}

// GetUser retrieves a user from the Marzban panel.
func (c *Client) GetUser(username string) (*UserResponse, error) {
	b, status, err := c.do("GET", "/api/user/"+username, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("get user failed: HTTP %d", status)
	}
	var u UserResponse
	json.Unmarshal(b, &u) //nolint:errcheck
	return &u, nil
}

// DisableUser disables a user on the Marzban panel.
func (c *Client) DisableUser(username string) error {
	_, status, err := c.do("PUT", "/api/user/"+username, map[string]string{"status": "disabled"})
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("disable user failed: HTTP %d", status)
	}
	return nil
}
