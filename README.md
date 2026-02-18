# annuminas

A Go CLI tool for managing Docker Hub repositories via the Docker Hub API. Named after the first capital of Arnor, seat of the palantir. Part of the `arnor` suite.

## Install

```bash
go install github.com/dukerupert/annuminas@latest
```

Or build from source:

```bash
go build -o annuminas .
```

## Configuration

Set your Docker Hub credentials in `~/.dotfiles/.env` or a local `.env` file:

```env
DOCKERHUB_USERNAME=your_username
DOCKERHUB_TOKEN=your_personal_access_token
```

Create a Personal Access Token at https://hub.docker.com/settings/security with read/write permissions.

## Usage

```bash
# Verify credentials
annuminas ping

# Repositories
annuminas repo list
annuminas repo get <name>
annuminas repo create --name myapp --private --description "My app"
annuminas repo delete --name myapp
annuminas repo ensure --name myapp

# Use --namespace flag for organization repos (defaults to DOCKERHUB_USERNAME)
annuminas repo list --namespace my-org
```

## Development

```bash
go run .              # Run without building
go test ./...         # Run all tests
go vet ./...          # Static analysis
gofmt -w .            # Format code
```
