# annuminas -- Project Scope

## Overview

`annuminas` is a lightweight CLI tool for managing Docker Hub repositories via the Docker Hub API. It is designed to be used standalone or as a component of the `arnor` infrastructure management suite.

Named after the first capital of Arnor, seat of the palantir -- the central registry where the king's records are kept.

## Repository

`github.com/dukerupert/annuminas`

## Status

**Planned**

## Technology

- **Language:** Go
- **CLI framework:** Cobra
- **Key dependencies:** `github.com/spf13/cobra`, `github.com/joho/godotenv`

## Configuration

Credentials are loaded from `~/.dotfiles/.env` with fallback to `.env` in the current directory.

Required environment variables:

```env
DOCKERHUB_USERNAME=your_username
DOCKERHUB_TOKEN=your_personal_access_token
```

> Use a Docker Hub Personal Access Token (PAT) rather than your account password. Create one at https://hub.docker.com/settings/security with read/write permissions.

## Project Structure

```
annuminas/
├── main.go
├── cmd/
│   ├── root.go         # env loading, cobra setup, client init
│   ├── repo.go         # repository subcommands
│   └── ping.go         # credential verification
├── pkg/
│   └── dockerhub/
│       └── client.go   # Docker Hub API client (importable by arnor)
├── .env.example
└── README.md
```

> **Important:** The client lives in `pkg/dockerhub/`, not `internal/`. This allows `arnor` to import it directly via a `replace` directive in `go.mod` during local development.

## Commands

### v1.0.0

| Command | Description |
|---|---|
| `ping` | Verify credentials by authenticating and confirming access |
| `repo list` | List all repositories in the user's namespace |
| `repo get` | Get details for a specific repository |
| `repo create` | Create a new repository (public by default) |
| `repo delete` | Delete a repository by name |
| `repo ensure` | Create a repository if it doesn't exist (idempotent) |

### v2.0.0 -- Quality of Life

- `--output json` flag for machine-readable output
- `--quiet` flag for scripting
- `--namespace` flag to operate on organization namespaces (defaults to `DOCKERHUB_USERNAME`)
- Shell autocompletion (bash, zsh, fish)
- `tag list` command to list tags for a repository

## Flags

### `repo create`

| Flag | Required | Default | Description |
|---|---|---|---|
| `--name` | yes | -- | Repository name (e.g. `myapp`) |
| `--private` | no | `false` | Whether the repository is private |
| `--description` | no | `""` | Short description for the repository |
| `--namespace` | no | `$DOCKERHUB_USERNAME` | Namespace (user or org) to create under |

### `repo list`

| Flag | Required | Default | Description |
|---|---|---|---|
| `--namespace` | no | `$DOCKERHUB_USERNAME` | Namespace to list repos from |
| `--page-size` | no | `25` | Number of results per page |

### `repo ensure`

| Flag | Required | Default | Description |
|---|---|---|---|
| `--name` | yes | -- | Repository name to ensure exists |
| `--namespace` | no | `$DOCKERHUB_USERNAME` | Namespace (user or org) |

> `repo ensure` is the primary command used by `arnor` during project creation. It is idempotent: if the repo already exists, it returns success without modification.

### `repo delete`

| Flag | Required | Default | Description |
|---|---|---|---|
| `--name` | yes | -- | Repository name to delete |
| `--namespace` | no | `$DOCKERHUB_USERNAME` | Namespace (user or org) |

## API Reference

Base URL: `https://hub.docker.com`

Authentication uses a two-step flow:

1. **Obtain a JWT** -- `POST /v2/auth/token` with `identifier` (username) and `secret` (PAT) in the request body. Returns a JSON object with a `token` field.
2. **Use the JWT** -- All subsequent requests include `Authorization: Bearer {token}` header.

Key endpoints:

```
POST   /v2/auth/token                                        # authenticate, get JWT
GET    /v2/namespaces/{namespace}/repositories                # list repos
POST   /v2/namespaces/{namespace}/repositories                # create repo
GET    /v2/namespaces/{namespace}/repositories/{name}         # get repo details
HEAD   /v2/namespaces/{namespace}/repositories/{name}         # check if repo exists
DELETE /v2/namespaces/{namespace}/repositories/{name}         # delete repo (v2.0.0)
GET    /v2/namespaces/{namespace}/repositories/{name}/tags    # list tags (v2.0.0)
```

### Authentication Request

```
POST /v2/auth/token
Content-Type: application/json

{
  "identifier": "username",
  "secret": "personal_access_token"
}
```

Response:

```json
{
  "token": "eyJhbG..."
}
```

### Create Repository Request

```
POST /v2/namespaces/{namespace}/repositories
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "myapp",
  "namespace": "myuser",
  "description": "",
  "is_private": false
}
```

Response: `201 Created`

### Check Repository Exists

```
HEAD /v2/namespaces/{namespace}/repositories/{name}
Authorization: Bearer {token}
```

Response: `200 OK` if exists, `404 Not Found` if not.

### List Repositories Response

```
GET /v2/namespaces/{namespace}/repositories?page=1&page_size=25
Authorization: Bearer {token}
```

Response includes paginated results with `count`, `next`, `previous`, and `results` array.

## Client API (pkg/dockerhub)

The `pkg/dockerhub` package exposes the following API for use by `arnor`:

```go
package dockerhub

// NewClient returns a Client authenticated with the given credentials.
// It does NOT authenticate immediately -- authentication is lazy on first API call.
func NewClient(username, token string) *Client

// Ping authenticates and verifies the credentials are valid.
func (c *Client) Ping() error

// ListRepos returns all repositories in the given namespace.
func (c *Client) ListRepos(namespace string) ([]Repository, error)

// GetRepo returns details for a specific repository.
func (c *Client) GetRepo(namespace, name string) (*Repository, error)

// RepoExists checks if a repository exists (HEAD request, no body parsed).
func (c *Client) RepoExists(namespace, name string) (bool, error)

// CreateRepo creates a new repository. Returns the created repository.
func (c *Client) CreateRepo(namespace, name string, isPrivate bool) (*Repository, error)

// EnsureRepo creates a repository if it doesn't exist. Idempotent.
func (c *Client) EnsureRepo(namespace, name string) error

// DeleteRepo deletes a repository by name.
func (c *Client) DeleteRepo(namespace, name string) error
```

### Repository Model

```go
type Repository struct {
    Name            string `json:"name"`
    Namespace       string `json:"namespace"`
    Description     string `json:"description"`
    IsPrivate       bool   `json:"is_private"`
    StarCount       int    `json:"star_count"`
    PullCount       int    `json:"pull_count"`
    LastUpdated     string `json:"last_updated"`
    DateRegistered  string `json:"date_registered"`
}
```

## Integration with arnor

When used as part of `arnor`, the `pkg/dockerhub` client is imported directly. The key use case in `arnor project create` is ensuring a Docker Hub repository exists before setting up CI/CD:

```go
import "github.com/dukerupert/annuminas/pkg/dockerhub"

client := dockerhub.NewClient(os.Getenv("DOCKERHUB_USERNAME"), os.Getenv("DOCKERHUB_TOKEN"))
err := client.EnsureRepo(username, projectName)
```

The docker image name is auto-derived as `{DOCKERHUB_USERNAME}/{projectName}`, eliminating any manual input.

### arnor go.mod integration

```
require github.com/dukerupert/annuminas v0.0.0

replace github.com/dukerupert/annuminas => ../annuminas
```

## Notes for Implementing Agents

- Authentication is a two-step flow: first obtain a JWT via `POST /v2/auth/token`, then use it as a Bearer token on all subsequent requests. The JWT has a short TTL -- cache it for the duration of a command but do not persist it.
- The `EnsureRepo` method is the most important for `arnor` integration -- it must be idempotent (check existence with HEAD first, create only if 404).
- Docker Hub API responses for list endpoints are paginated with `count`, `next`, `previous`, `results` fields. Implement pagination handling in `ListRepos`.
- The namespace defaults to the authenticated username but can be an organization name for org-owned repos.
- Error responses from Docker Hub return JSON with a `message` field -- parse and surface these in error messages.
- Mirror the structure and conventions of `fornost` as closely as possible for consistency across the suite:
  - Same `doRequest` / `get` / `post` / `delete` HTTP helper pattern
  - Same `godotenv` loading in `cmd/root.go`
  - Same Cobra command structure
  - `pkg/` (not `internal/`) so `arnor` can import the client
- The `ping` command should call `POST /v2/auth/token` and confirm a valid JWT is returned.
- Include a `.env.example` with placeholder values for `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.
