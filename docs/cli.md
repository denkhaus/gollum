# CLI Documentation

## Usage

```bash
gollum [command] [arguments]
```

## Commands

### Running Gollum (Default)

```bash
gollum
```

Runs the default application. If a default flow exists at `.gollum/flows/default/main.xml`,
it will be executed. Otherwise, the TUI interface starts.

### ACP Command

#### gollum acp

Start Gollum ACP server (Agent Client Protocol) for AI agent communication.

```bash
gollum acp [flags]
```

**Flags:**
- `-t, --transport <type>` - Transport type: `stdio` or `http` (default: `stdio`)
- `--host <address>` - HTTP server bind address (default: `0.0.0.0`)
- `--port <number>` - HTTP server port (default: `8080`)
- `--shutdown-timeout <duration>` - HTTP server graceful shutdown timeout (default: `10s`)

**Environment Variables:**
- `GOLLUM_ACP_TRANSPORT` - Override transport type
- `GOLLUM_ACP_HOST` - Override HTTP server bind address
- `GOLLUM_ACP_PORT` - Override HTTP server port
- `GOLLUM_ACP_SHUTDOWN_TIMEOUT` - Override shutdown timeout

**Transport Modes:**

1. **Stdio Mode (default)** - Direct stdin/stdout communication
   - Best for: Local development, single-client scenarios
   - Usage: `gollum acp` or `gollum acp --transport stdio`
   - Communication: Standard input/output streams

2. **HTTP Mode** - HTTP + Server-Sent Events
   - Best for: Multi-client servers, production deployments
   - Usage: `gollum acp --transport http --host 0.0.0.0 --port 8080`
   - Communication: HTTP endpoints with SSE for real-time streaming
   - Supports: Multiple concurrent ACP sessions

**Examples:**

```bash
# Local development with stdio (default)
gollum acp

# Multi-client HTTP server
gollum acp --transport http --host 0.0.0.0 --port 8080

# Production deployment with custom timeout
gollum acp -t http --host 0.0.0.0 --port 9000 --shutdown-timeout 30s

# Using environment variables
export GOLLUM_ACP_TRANSPORT=http
export GOLLUM_ACP_PORT=9000
gollum acp
```

### Flow Commands

#### gollum flow lint

Validate a flow file or module directory.

```bash
gollum flow lint <path>
```

**Flags:**
- `-v, --verbose` - Enable verbose output
- `-o, --output` - Output format (text, json)

**Examples:**
```bash
gollum flow lint .gollum/flows/examples/simple-flow.xml
gollum flow lint .gollum/flows/my-module/
gollum flow lint --verbose .gollum/flows/complex-flow.xml
gollum flow lint --output json .gollum/flows/my-module/
```

#### gollum flow run

Execute a flow.

```bash
gollum flow run <path>
```

**Flags:**
- `-v, --verbose` - Enable verbose output

**Examples:**
```bash
gollum flow run .gollum/flows/examples/simple-flow.xml
gollum flow run .gollum/flows/my-module/
gollum flow run --verbose .gollum/flows/complex-flow.xml
```

## Default Flows

To create a default flow that runs when you type `gollum`:

1. Create a flow module at `.gollum/flows/default/main.xml`
2. This flow will execute automatically when running `gollum` with no arguments
3. Workspace-local flows override global flows (`~/.config/gollum/flows/default/main.xml`)

## Exit Codes

- `0`: Success
- `1`: Error (validation failed, execution error)
- `2`: Usage error (invalid arguments)
