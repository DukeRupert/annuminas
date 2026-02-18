package dockerhub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "https://hub.docker.com"

// Client wraps the Docker Hub API v2.
type Client struct {
	Username   string
	Token      string
	jwt        string
	httpClient *http.Client
}

// NewClient returns a Client authenticated with the given credentials.
// It does NOT authenticate immediately — authentication is lazy on first API call.
func NewClient(username, token string) *Client {
	return &Client{
		Username:   username,
		Token:      token,
		httpClient: &http.Client{},
	}
}

// --- Authentication ---

// authenticate obtains a JWT from Docker Hub using the two-step auth flow.
func (c *Client) authenticate() error {
	if c.jwt != "" {
		return nil
	}

	payload := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{Username: c.Username, Password: c.Token}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode auth request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v2/users/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read auth response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr errorResponse
		if json.Unmarshal(raw, &apiErr) == nil && apiErr.Message != "" {
			return fmt.Errorf("authentication failed: %s", apiErr.Message)
		}
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	var authResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &authResp); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}

	c.jwt = authResp.Token
	return nil
}

// --- HTTP helpers ---

func (c *Client) doRequest(method, endpoint string, body io.Reader, out any) (int, error) {
	if err := c.authenticate(); err != nil {
		return 0, err
	}

	req, err := http.NewRequest(method, baseURL+endpoint, body)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr errorResponse
		if json.Unmarshal(raw, &apiErr) == nil && apiErr.Message != "" {
			return resp.StatusCode, fmt.Errorf("docker hub api error: %s", apiErr.Message)
		}
		return resp.StatusCode, fmt.Errorf("docker hub api error: status %d", resp.StatusCode)
	}

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func (c *Client) get(endpoint string, out any) error {
	_, err := c.doRequest(http.MethodGet, endpoint, nil, out)
	return err
}

func (c *Client) post(endpoint string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	_, err = c.doRequest(http.MethodPost, endpoint, bytes.NewReader(body), out)
	return err
}

func (c *Client) delete(endpoint string) error {
	_, err := c.doRequest(http.MethodDelete, endpoint, nil, nil)
	return err
}

func (c *Client) head(endpoint string) (int, error) {
	if err := c.authenticate(); err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodHead, baseURL+endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// getAll fetches all pages from a paginated Docker Hub endpoint.
// Docker Hub uses {"count", "next", "previous", "results"} format.
func getAll[T any](c *Client, endpoint string) ([]T, error) {
	var all []T
	page := 1

	for {
		url := fmt.Sprintf("%s?page=%d&page_size=50", endpoint, page)

		var envelope struct {
			Count    int     `json:"count"`
			Next     *string `json:"next"`
			Previous *string `json:"previous"`
			Results  []T     `json:"results"`
		}
		if err := c.get(url, &envelope); err != nil {
			return nil, err
		}

		all = append(all, envelope.Results...)

		if envelope.Next == nil {
			break
		}
		page++
	}

	return all, nil
}

// --- API models ---

type errorResponse struct {
	Message string `json:"message"`
}

// Repository represents a Docker Hub repository.
type Repository struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Description    string `json:"description"`
	IsPrivate      bool   `json:"is_private"`
	StarCount      int    `json:"star_count"`
	PullCount      int    `json:"pull_count"`
	LastUpdated    string `json:"last_updated"`
	DateRegistered string `json:"date_registered"`
}

// --- API methods ---

// Ping authenticates and verifies the credentials are valid.
func (c *Client) Ping() error {
	return c.authenticate()
}

// ListRepos returns all repositories in the given namespace.
func (c *Client) ListRepos(namespace string) ([]Repository, error) {
	return getAll[Repository](c, fmt.Sprintf("/v2/namespaces/%s/repositories", namespace))
}

// GetRepo returns details for a specific repository.
func (c *Client) GetRepo(namespace, name string) (*Repository, error) {
	var repo Repository
	if err := c.get(fmt.Sprintf("/v2/namespaces/%s/repositories/%s", namespace, name), &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// RepoExists checks if a repository exists (HEAD request, no body parsed).
func (c *Client) RepoExists(namespace, name string) (bool, error) {
	status, err := c.head(fmt.Sprintf("/v2/namespaces/%s/repositories/%s", namespace, name))
	if err != nil {
		return false, err
	}
	return status == http.StatusOK, nil
}

// CreateRepo creates a new repository. Returns the created repository.
func (c *Client) CreateRepo(namespace, name, description string, isPrivate bool) (*Repository, error) {
	payload := struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		Description string `json:"description"`
		IsPrivate   bool   `json:"is_private"`
	}{
		Name:        name,
		Namespace:   namespace,
		Description: description,
		IsPrivate:   isPrivate,
	}

	var repo Repository
	if err := c.post(fmt.Sprintf("/v2/namespaces/%s/repositories", namespace), payload, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// EnsureRepo creates a repository if it doesn't exist. Idempotent.
func (c *Client) EnsureRepo(namespace, name string) error {
	exists, err := c.RepoExists(namespace, name)
	if err != nil {
		return fmt.Errorf("check repo existence: %w", err)
	}
	if exists {
		return nil
	}

	_, err = c.CreateRepo(namespace, name, "", false)
	if err != nil {
		return fmt.Errorf("create repo: %w", err)
	}
	return nil
}

// DeleteRepo deletes a repository by name.
func (c *Client) DeleteRepo(namespace, name string) error {
	return c.delete(fmt.Sprintf("/v2/namespaces/%s/repositories/%s", namespace, name))
}
