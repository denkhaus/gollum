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
