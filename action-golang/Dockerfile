# Custom Go Image for Claude Code GitHub/Forgejo Actions
# Based on golang:1.25-alpine with pre-installed development tools

FROM golang:1.25-alpine

# Build arguments for versioning
ARG GOLANGCI_LINT_VERSION=v2.6.1
ARG GH_VERSION=v2.60.0

# Install system dependencies and tools
RUN apk add --no-cache \
    tzdata \
    curl \
    jq \
    git \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Set timezone to Europe/Berlin
RUN ln -sf /usr/share/zoneinfo/Europe/Berlin /etc/localtime && \
    echo "Europe/Berlin" > /etc/timezone

# Install GitHub CLI (gh)
RUN curl -L "https://github.com/cli/cli/releases/download/${GH_VERSION}/gh_${GH_VERSION#v}_linux_amd64.tar.gz" -o /tmp/gh.tar.gz && \
    tar -xzf /tmp/gh.tar.gz -C /tmp && \
    mv /tmp/gh_${GH_VERSION#v}_linux_amd64/bin/gh /usr/local/bin/ && \
    rm -rf /tmp/gh.tar.gz /tmp/gh_${GH_VERSION#v}_linux_amd64

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b /usr/local/bin "${GOLANGCI_LINT_VERSION}"

# Install mockgen (uber mockgen)
RUN go install go.uber.org/mock/mockgen@latest

# Set environment variables
ENV TZ=Europe/Berlin
ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin

# Verify installations
RUN echo "Verifying installations..." && \
    go version && \
    git --version && \
    gh --version && \
    golangci-lint --version && \
    mockgen -version || echo "mockgen version check failed"

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/bin/sh"]
