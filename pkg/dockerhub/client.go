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
	Username string
	Password string
	jwt      string
	http     *http.Client
}

// NewClient returns a Client authenticated with the given credentials.
// It does NOT authenticate immediately — authentication is lazy on first API call.
func NewClient(username, password string) *Client {
	return &Client{
		Username: username,
		Password: password,
		http:     &http.Client{},
	}
}

// --- Authentication ---

// authenticate obtains a JWT from Docker Hub via POST /v2/auth/token.
func (c *Client) authenticate() error {
	if c.jwt != "" {
		return nil
	}

	payload := struct {
		Identifier string `json:"identifier"`
		Secret     string `json:"secret"`
	}{Identifier: c.Username, Secret: c.Password}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode auth request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v2/auth/token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
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
		if json.Unmarshal(raw, &apiErr) == nil && apiErr.text() != "" {
			return fmt.Errorf("authentication failed: %s", apiErr.text())
		}
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &authResp); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}

	c.jwt = authResp.AccessToken
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

	resp, err := c.http.Do(req)
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
		if json.Unmarshal(raw, &apiErr) == nil && apiErr.text() != "" {
			return resp.StatusCode, fmt.Errorf("docker hub api error (%s %s, status %d): %s\nresponse body: %s",
				method, endpoint, resp.StatusCode, apiErr.text(), string(raw))
		}
		return resp.StatusCode, fmt.Errorf("docker hub api error (%s %s, status %d)\nresponse body: %s",
			method, endpoint, resp.StatusCode, string(raw))
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

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Surface auth/server errors so callers don't misinterpret 401 as "not found".
	// 404 is allowed through — RepoExists relies on it to mean "doesn't exist".
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return resp.StatusCode, fmt.Errorf("docker hub api error: status %d", resp.StatusCode)
	}

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
	Detail  string `json:"detail"`
}

func (e errorResponse) text() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Detail
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

// AccessToken represents a Docker Hub personal access token.
type AccessToken struct {
	UUID        string   `json:"uuid"`
	TokenLabel  string   `json:"token_label"`
	Scopes      []string `json:"scopes"`
	IsActive    bool     `json:"is_active"`
	Token       string   `json:"token"`
	CreatedAt   string   `json:"created_at"`
	LastUsed    string   `json:"last_used"`
	GeneratedBy string   `json:"generated_by"`
	CreatorIP   string   `json:"creator_ip"`
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

// CreateAccessToken creates a personal access token with the given label and scopes.
// Valid scopes: "repo:admin", "repo:write", "repo:read", "repo:public_read".
// The token value is only available in the response from creation — it cannot be retrieved later.
func (c *Client) CreateAccessToken(label string, scopes []string) (*AccessToken, error) {
	payload := struct {
		TokenLabel string   `json:"token_label"`
		Scopes     []string `json:"scopes"`
	}{
		TokenLabel: label,
		Scopes:     scopes,
	}

	var token AccessToken
	if err := c.post("/v2/access-tokens", payload, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// ListAccessTokens returns all personal access tokens for the authenticated user.
func (c *Client) ListAccessTokens() ([]AccessToken, error) {
	return getAll[AccessToken](c, "/v2/access-tokens")
}

// DeleteAccessToken deletes a personal access token by UUID.
func (c *Client) DeleteAccessToken(uuid string) error {
	return c.delete(fmt.Sprintf("/v2/access-tokens/%s", uuid))
}
