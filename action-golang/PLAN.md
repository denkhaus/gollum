# Development Plan: Golang Claude Code Action

## Current State (Phase 1)

### What we have:
- **Dockerfile**: Base image with Go 1.25 Alpine + pre-installed tools
  - golangci-lint v2.6.1
  - GitHub CLI (gh) v2.60.0
  - mockgen (uber)
  - Europe/Berlin timezone
- **Makefile**: Build, test, push automation
- **Workflow files**: `.github/workflows/claude-code.yaml` and `.forgejo/workflows/claude-code-forgejo.yaml`
  - ~300 lines of YAML
  - All logic embedded in workflow steps

### Problems:
- Every workflow run installs dependencies (SLOW)
- 300+ lines of YAML for simple logic
- Hard to test locally
- Duplication between GitHub and Forgejo workflows

---

## Phase 2: Current Task

### Goals:
1. Build custom Docker image: `denkhaus/golang-claude-action`
2. Test image locally
3. Push to Docker Hub
4. Update workflows to use custom image

### Steps:
```bash
# Build and test
make build
make test

# Login and push
docker login
make push

# Update workflow files
# Change: image: golang:1.25-alpine
# To: image: denkhaus/golang-claude-action:latest
```

### Expected outcome:
- Workflow runs ~30-60 seconds faster (no install step)
- Remove "Install dependencies" step from workflows

---

## Phase 3: Entrypoint Script (Future Enhancement)

### Goals:
- Move workflow logic into Docker image
- Simplify workflows to ~50 lines
- Better testability

### Implementation:

#### 1. Create `/action-golang/entrypoint.sh`
```bash
#!/bin/sh
set -e

# Parse GitHub event from env var
# Load issue context if needed
# Extract prompt
# Write prompt to /tmp/claude-prompts/prompt.txt
# Set outputs for GitHub Actions
```

#### 2. Update Dockerfile
```dockerfile
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD ["run"]
```

#### 3. Simplified workflow becomes:
```yaml
jobs:
  claude:
    runs-on: ubuntu-latest
    container: denkhaus/golang-claude-action:latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Claude Code
        run: /entrypoint.sh prepare
        env:
          GITHUB_EVENT: ${{ toJSON(github.event) }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
      - name: Run Claude Code Action
        uses: anthropics/claude-code-base-action@main
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          prompt_file: /tmp/claude-prompts/prompt.txt
      - name: Handle results
        run: /entrypoint.sh handle-results
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Benefits:
- **50 lines of YAML** instead of 300
- **Easy local testing**: `docker run -e GITHUB_EVENT=... denkhaus/golang-claude-action`
- **Single source of truth** for GitHub and Forgejo
- **Better error handling** in bash than YAML

---

## Phase 4: Advanced Features (Optional)

### Ideas:
1. **Go-based entrypoint** instead of bash
   - Better error handling
   - Type safety
   - Easier to maintain

2. **Configuration via environment variables**
   ```yaml
   env:
     CLAUDE_MODEL: claude-sonnet-4-20250514
     CLAUDE_MAX_TOKENS: 200000
     CLAUDE_TEMPERATURE: 0.0
   ```

3. **Built-in skills support**
   - Bundle common skills in image
   - Load from `/skills` directory

4. **Multi-platform support**
   ```bash
   docker buildx build --platform linux/amd64,linux/arm64 ...
   ```

---

## Versioning Strategy

### Image tags:
- `latest` - Most recent stable build
- `v1.0.0`, `v1.1.0` - Semantic versioning
- `20250103` - Date-based for frequent updates

### Breaking changes:
- Update major version (v1 → v2)
- Update workflows to use new tag

---

## Testing Strategy

### Local testing:
```bash
# Test image
make test

# Interactive shell
make shell

# Test with real event (from file)
docker run -v $PWD/test-event.json:/tmp/event.json \
  -e GITHUB_EVENT_PATH=/tmp/event.json \
  denkhaus/golang-claude-action
```

### CI testing:
- Test workflow on fork/branch first
- Use `--dry-run` mode if available
- Check reaction emojis (eyes → +1/-1)

---

## Migration Path

### From current to Phase 2:
1. Build and push image
2. Update `container:` in both workflows
3. Remove "Install dependencies" step
4. Test on a comment/issue

### From Phase 2 to Phase 3:
1. Create entrypoint.sh
2. Test locally extensively
3. Update Dockerfile with new entrypoint
4. Build and push new image tag
5. Update workflows to use entrypoint
6. Remove 200+ lines of YAML
