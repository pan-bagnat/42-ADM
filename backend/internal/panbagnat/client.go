package panbagnat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// User represents the subset of Pan-Bagnat user data we care about.
type User struct {
	ID      string `json:"id"`
	FtLogin string `json:"ft_login"`
}

type listUsersResponse struct {
	Users         []User `json:"users"`
	NextPageToken string `json:"next_page_token"`
}

// Client wraps HTTP access to the Pan-Bagnat API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client using the provided base URL (e.g. https://pan-bagnat.local).
func NewClient(baseURL string) *Client {
	trimmed := strings.TrimSuffix(baseURL, "/")
	return &Client{
		baseURL: trimmed,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) isConfigured() bool {
	return c != nil && c.baseURL != ""
}

// ListAllUsers fetches every user visible to the caller by following pagination.
// The provided authorization header is forwarded to the Pan-Bagnat API.
func (c *Client) ListAllUsers(ctx context.Context, authHeader string) ([]User, error) {
	if !c.isConfigured() {
		return nil, errors.New("pan bagnat api base url not configured")
	}

	endpoint, err := url.Parse(c.baseURL + "/api/v1/admin/users")
	if err != nil {
		return nil, fmt.Errorf("parse users endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("limit", "200")
	endpoint.RawQuery = query.Encode()

	var all []User
	nextToken := ""

	for {
		reqURL := *endpoint
		if nextToken != "" {
			q := reqURL.Query()
			q.Set("next_page_token", nextToken)
			reqURL.RawQuery = q.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create users request: %w", err)
		}
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request users: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("pan bagnat users request failed: %s", resp.Status)
		}

		var payload listUsersResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode users response: %w", err)
		}
		resp.Body.Close()

		for _, user := range payload.Users {
			if user.FtLogin != "" {
				all = append(all, user)
			}
		}

		if payload.NextPageToken == "" {
			break
		}
		nextToken = payload.NextPageToken
	}

	return all, nil
}
