# Golang Claude Code Action - Custom Docker Image

A highly optimized, general-purpose Docker image for Claude Code GitHub/Forgejo Actions with pre-installed Go development tools.

## Overview

This custom Docker image significantly speeds up GitHub/Forgejo workflows by pre-installing all necessary tools. Instead of wasting 30-60 seconds for installations on every run, all tools are already included in the image.

While originally developed for Claude Code workflows, this image is designed to be **platform-agnostic** and can be used for any Go-based CI/CD pipeline on GitHub Actions, Forgejo, GitLab, or any Docker-based runner system.

### Included Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25 (Debian) | Go compiler and standard library |
| golangci-lint | v2.6.1 | Go linter with many rules |
| GitHub CLI (gh) | v2.60.0 | GitHub API interaction |
| Mage | latest | Make/sh replacement in Go |
| mockgen (uber) | latest | Mock generation for tests |
| Bun | latest | JavaScript package manager & runner |
| git | - | Version control |
| jq | - | JSON processor |
| curl | - | HTTP client |
| bash | - | Shell |

### System Settings

- **Base Image**: `golang:1.25` (Debian Bookworm)
- **Timezone**: Europe/Berlin
- **Working Directory**: `/workspace`

## Quick Start

### Using the image locally

```bash
# Pull the image
docker pull denkhaus/golang-claude-action:latest

# Start interactive shell
docker run --rm -it denkhaus/golang-claude-action /bin/sh

# Run Go command
docker run --rm denkhaus/golang-claude-action go version
```

### In GitHub Actions

```yaml
jobs:
  my-job:
    runs-on: ubuntu-latest
    container:
      image: denkhaus/golang-claude-action:latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Go tests
        run: go test ./...
      - name: Run linter
        run: golangci-lint run
```

### In Forgejo Actions

```yaml
jobs:
  my-job:
    runs-on: docker
    container:
      image: denkhaus/golang-claude-action:latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Go tests
        run: go test ./...
```

### In GitLab CI

```yaml
job:
  image: denkhaus/golang-claude-action:latest
  script:
    - go test ./...
    - go build ./...
```

## Development with Mage

This project uses [Mage](https://magefile.org/) as a build tool instead of Make. Mage is a Make/sh replacement written in Go.

### Install Mage

```bash
# macOS/Linux
curl -sL https://magefile.org/install.sh | bash

# Or via Go
go install github.com/magefile/mage@latest
```

### Available Targets

```bash
# List all targets
mage -l

# Build image
mage build

# Push image to registry
mage push

# Test image locally
mage test

# Start interactive shell in image
mage shell

# Clean up dangling images
mage clean

# Build and push release with date tag
mage release

# Show help
mage list
```

### Mage Targets

| Target | Description |
|--------|-------------|
| `build` | Build Docker image (default) |
| `push` | Push image to Docker Hub |
| `test` | Test image locally (verify all tools) |
| `shell` | Start interactive shell in container |
| `login` | Login to Docker registry |
| `clean` | Clean up dangling Docker images |
| `release` | Build release with date tag (YYYYMMDD) |
| `list` | Show detailed help |

## Project Structure

```
action-golang/
├── Dockerfile           # Docker image definition
├── magefiles/           # Mage build scripts
│   ├── mage.go         # Main targets
│   ├── config.go       # Configuration
│   ├── colors.go       # Colored output
│   ├── docker.go       # Docker operations
│   ├── io.go           # I/O streams
│   └── go.mod          # Go dependencies
├── PLAN.md             # Development roadmap
└── README.md           # This file
```

### Mage Module Architecture

The magefile is modularly structured for easy extension:

- **config.go**: Configuration structs (Image, Registry, BuildArgs)
- **colors.go**: Printer for colored terminal output
- **docker.go**: DockerClient encapsulates all Docker operations
- **io.go**: Global I/O streams (for testability)
- **mage.go**: Public targets

Easily extensible with new modules like:
- `git.go` for Git operations
- `prompt.go` for prompt extraction in workflows
- `github.go` for GitHub API operations
- `forgejo.go` for Forgejo API operations

## Image Tags

- `latest` - Latest stable version
- `YYYYMMDD` - Date-based releases (e.g. `20250103`)
- `vX.Y.Z` - Semantic versioning (future)

## Configuration

### Environment Variables in Image

- `TZ=Europe/Berlin` - Timezone
- `PATH=/go/bin:/usr/local/go/bin:/usr/local/bin:/github/home/.bun/bin`

### Build Args

```dockerfile
ARG GOLANGCI_LINT_VERSION=v2.6.1
ARG GH_VERSION=v2.60.0
ARG NODE_VERSION=22
ARG BUN_VERSION=1.1.38
```

## Local Development

### Build image with custom versions

```bash
mage build
# Or with custom args via Docker:
docker build \
  --build-arg GOLANGCI_LINT_VERSION=v2.7.0 \
  --build-arg GH_VERSION=v2.61.0 \
  -t my-golang-action:latest .
```

### Integrate in CI

```yaml
- name: Run in custom container
  uses: actions/checkout@v4
- name: Run Go tests
  run: |
    go test ./...
    go build ./...
```

## Benefits of Custom Image

1. **Speed**: Save 30-60 seconds per workflow run
2. **Consistency**: Always same tool versions
3. **Platform Independence**: Works on GitHub, Forgejo, GitLab, any Docker-based runner
4. **Fewer Dependencies**: No setup actions needed
5. **Better Testability**: Same environment locally as in CI
6. **Reusability**: Can be used for any Go project, not just Claude Code workflows

## Use Cases

This image is suitable for:

- **Claude Code Workflows**: Responding to issues, PRs, comments with AI assistance
- **Go CI/CD Pipelines**: Building, testing, linting Go projects
- **Multi-language Projects**: Go + JavaScript/TypeScript with Bun
- **GitHub Automation**: Using gh CLI for automation tasks
- **General Go Development**: Any Go-based workflow requiring tools

## Troubleshooting

### Bun not found

Bun is installed at `/github/home/.bun/bin/bun` - this is the path expected by `claude-code-base-action`. There are also symlinks in `/usr/local/bin/`. The image uses Debian with glibc (not Alpine with musl) for full Bun compatibility.

### Timezone issues

The image uses `Europe/Berlin`. For different timezones:

```dockerfile
RUN ln -sf /usr/share/zoneinfo/YOUR_TIMEZONE /etc/localtime
ENV TZ=YOUR_TIMEZONE
```

### Image too large

Alpine base image keeps it small (~200-300MB). For even smaller images, unnecessary tools could be removed.

## Roadmap

See [PLAN.md](PLAN.md) for planned enhancements including:
- Go-based entrypoint script for workflow automation
- Multi-platform support (linux/amd64, linux/arm64)
- Built-in GitHub/Forgejo API helpers
- Prompt extraction utilities

## License

MIT License - See LICENSE file for details.

## Contributing

Contributions are welcome! This project is designed to be general-purpose and extensible.

## Further Reading

- [Mage Documentation](https://magefile.org/)
- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Forgejo Actions Docs](https://forgejo.org/docs/2023.10/user/actions/)
- [GitLab CI Docs](https://docs.gitlab.com/ee/ci/)
- [Dockerfile Reference](https://docs.docker.com/engine/reference/builder/)
- [Claude Code](https://claude.ai/code)
