# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`annuminas` is a Go CLI tool for managing Docker Hub repositories via the Docker Hub API. It runs standalone or as an importable library used by the `arnor` infrastructure management suite. Named after the first capital of Arnor, seat of the palantir.

## Tech Stack

- **Language:** Go
- **CLI framework:** Cobra (`github.com/spf13/cobra`)
- **Env loading:** `github.com/joho/godotenv`
- **Config:** Credentials from `~/.dotfiles/.env` with fallback to `.env`; requires `DOCKERHUB_USERNAME` and `DOCKERHUB_PASSWORD`

## Build & Development Commands

```bash
go build -o annuminas .          # Build binary
go run .                         # Run without building
go test ./...                    # Run all tests
go test ./pkg/dockerhub/         # Run tests for a specific package
go vet ./...                     # Static analysis
gofmt -w .                       # Format code
```

## Architecture

```
main.go              # Entry point, calls cmd.Execute()
cmd/
  root.go            # Env loading, cobra setup, client init
  repo.go            # repo list, repo get, repo create, repo delete, repo ensure
  token.go           # token create, token list, token delete (PAT management)
  ping.go            # Credential verification via POST /v2/auth/token
pkg/
  dockerhub/
    client.go        # HTTP client wrapping Docker Hub API v2
```

- **cmd/** — Each file registers subcommands on the root cobra command. `root.go` handles env loading (godotenv) and initializes the shared Docker Hub client.
- **pkg/dockerhub/** — API client used by commands. Lives in `pkg/` (not `internal/`) so `arnor` can import it directly via a `replace` directive in `go.mod`.

## Key Design Decisions

- Authentication is two-step: `POST /v2/auth/token` returns a JWT, then all requests use `Authorization: Bearer {token}`. JWT is cached per command invocation, never persisted.
- API base: `https://hub.docker.com`
- `EnsureRepo` is the primary command for `arnor` integration — must be idempotent (HEAD to check existence, create only on 404).
- List endpoints are paginated (`count`, `next`, `previous`, `results`) — `ListRepos` must handle pagination.
- Namespace defaults to `DOCKERHUB_USERNAME` but can be an organization name.
- Error responses return JSON with a `message` or `detail` field — parse and surface these.
- Password auth (not PAT) is required for write operations (repo create, token management). PAT-issued JWTs are restricted by Docker Hub.
- Mirror structure and conventions of sibling project `fornost`: same `doRequest`/`get`/`post`/`delete` HTTP helper pattern, same godotenv loading, same Cobra command structure.
