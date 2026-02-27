# Annuminas

A CLI tool for managing Docker Hub repositories and personal access tokens via the Docker Hub API. Named after the first capital of Arnor, seat of the palantir. Part of the `arnor` infrastructure management suite.

Annuminas can create repos, manage their lifecycle, and provision scoped personal access tokens (PATs) — useful for setting up CI/CD pipelines that push container images to Docker Hub.

## Prerequisites

- Go 1.25+
- A Docker Hub account with username and password

## Installation

```bash
go install github.com/dukerupert/annuminas@latest
```

Or build from source:

```bash
git clone https://github.com/dukerupert/annuminas.git
cd annuminas
go build -o annuminas .
```

## Configuration

Annuminas reads credentials from environment variables. It loads a `.env` file automatically, checking `~/.dotfiles/.env` first with a fallback to `.env` in the current directory.

| Variable | Required | Description |
|---|---|---|
| `DOCKERHUB_USERNAME` | Yes | Your Docker Hub username |
| `DOCKERHUB_PASSWORD` | Yes | Your Docker Hub account password |

Example `.env` file:

```env
DOCKERHUB_USERNAME=myuser
DOCKERHUB_PASSWORD=mypassword
```

**Why password instead of a PAT?** Docker Hub restricts JWTs issued from personal access tokens — they cannot create repositories or manage other tokens. Password-based auth grants full API access, which is required for annuminas' core operations.

## Usage

### Verify credentials

```bash
annuminas ping
```

### Repositories

```bash
# List all repositories
annuminas repo list

# List repos in a specific org namespace
annuminas repo list --namespace myorg

# Get details for a repo
annuminas repo get my-app

# Create a repo
annuminas repo create --name my-app --description "My application" --private

# Create a repo only if it doesn't already exist (idempotent)
annuminas repo ensure --name my-app

# Delete a repo
annuminas repo delete --name my-app
```

### Personal Access Tokens

```bash
# Create a token with write access (includes read)
annuminas token create --label "ci-my-app" --scopes repo:write

# Create a read-only token
annuminas token create --label "readonly" --scopes repo:read

# List all tokens
annuminas token list

# Delete a token by UUID
annuminas token delete --uuid <uuid>
```

Available scopes (higher scopes include lower ones):

| Scope | Grants |
|---|---|
| `repo:public_read` | Pull public images |
| `repo:read` | Pull any image |
| `repo:write` | Push and pull images |
| `repo:admin` | Full repo management |

**The token value is only shown at creation time and cannot be retrieved later.** Save it immediately.

### Typical workflow

Set up a new service for CI/CD deployment:

```bash
# Create the repo
annuminas repo create --name my-service

# Create a scoped token for CI to push images
annuminas token create --label "ci-my-service" --scopes repo:write
# Output:
#   Token: dckr_pat_xxxxx
#   Save this token now — it cannot be retrieved later.
```

Use the returned PAT as `DOCKERHUB_TOKEN` in your CI/CD pipeline (e.g. GitHub Actions secrets).

## Library Usage

The `pkg/dockerhub` package can be imported directly by other Go projects:

```go
import "github.com/dukerupert/annuminas/pkg/dockerhub"

client := dockerhub.NewClient("username", "password")

// Create a repo
repo, err := client.CreateRepo("myuser", "my-app", "description", false)

// Ensure a repo exists (idempotent)
err = client.EnsureRepo("myuser", "my-app")

// Create a PAT
token, err := client.CreateAccessToken("ci-token", []string{"repo:write"})
fmt.Println(token.Token) // save this immediately
```

## Development

```bash
go run .              # Run without building
go test ./...         # Run all tests
go vet ./...          # Static analysis
gofmt -w .            # Format code
```
