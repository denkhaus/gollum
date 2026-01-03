# Custom Go Image for Claude Code GitHub/Forgejo Actions
# Based on golang:1.25 (Debian) with pre-installed development tools
# Includes Node.js and Bun for claude-code-base-action compatibility

FROM golang:1.25

# Build arguments for versioning
ARG GOLANGCI_LINT_VERSION=v2.6.1
ARG GH_VERSION=v2.60.0
ARG NODE_VERSION=22
ARG BUN_VERSION=1.1.38

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies and tools
RUN apt-get update && apt-get install -y \
    tzdata \
    curl \
    jq \
    git \
    ca-certificates \
    bash \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Set timezone to Europe/Berlin
RUN ln -sf /usr/share/zoneinfo/Europe/Berlin /etc/localtime && \
    echo "Europe/Berlin" > /etc/timezone

# Install GitHub CLI (gh)
RUN curl -L "https://github.com/cli/cli/releases/download/${GH_VERSION}/gh_${GH_VERSION#v}_linux_amd64.tar.gz" -o /tmp/gh.tar.gz && \
    tar -xzf /tmp/gh.tar.gz -C /tmp && \
    mv /tmp/gh_${GH_VERSION#v}_linux_amd64/bin/gh /usr/local/bin/ && \
    rm -rf /tmp/gh.tar.gz /tmp/gh_${GH_VERSION#v}_linux_amd64

# Install Bun (needed for claude-code-base-action)
# Install directly to /github/home/.bun/bin where the action expects it
# Note: Bun requires glibc (Debian), not musl (Alpine)
RUN mkdir -p /github/home/.bun/bin && \
    curl -fsSL https://bun.sh/install | BUN_INSTALL=/github/home/.bun bash && \
    /github/home/.bun/bin/bun --version && \
    # Also symlink to /usr/local/bin for general use
    ln -sf /github/home/.bun/bin/bun /usr/local/bin/bun && \
    ln -sf /github/home/.bun/bin/bun /usr/local/bin/bunx

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b /usr/local/bin "${GOLANGCI_LINT_VERSION}"

# Install mockgen (uber mockgen)
RUN go install go.uber.org/mock/mockgen@latest

# Install Mage (make/sh replacement in Go)
RUN go install github.com/magefile/mage@latest

# Set environment variables
ENV TZ=Europe/Berlin
ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin:/github/home/.bun/bin

# Verify installations
RUN echo "Verifying installations..." && \
    go version && \
    bun --version && \
    git --version && \
    gh --version && \
    golangci-lint --version && \
    mage --version && \
    mockgen -version || echo "mockgen version check failed"

# Copy Magefiles to a separate location that won't be overwritten by repository checkouts
# These are the action-golang specific magefiles for GitHub event handling
# We need to copy go.mod and go.sum together with the magefiles
COPY magefiles/ /opt/magefiles/
COPY go.mod /opt/
COPY go.sum /opt/

# Verify Mage is working and can find the magefiles in /opt/magefiles
WORKDIR /opt
RUN mage -l

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/bin/bash"]
