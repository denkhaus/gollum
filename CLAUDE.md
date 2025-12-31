# Gollum Project - Development Guidelines

## Mock Generation with MockGen

When working with interfaces in tests, use the centralized mock generation approach.
Maintain a centralized `pkg/mocks/generate.go` file where all mocks generation commands live


### Use Centralized Mocks
Import and use mocks from the centralized location:
```go
import "github.com/denkhaus/gollum/pkg/mocks"

ctrl := gomock.NewController(t)
mockRegistry := mocks.NewMockAgentRegistry(ctrl)
```

### Benefits
- Single source of truth for all mocks
- Reusable across all packages
- Easy regeneration with go generate
- Consistent mock patterns

## Project Structure

- `pkg/registry/` - Agent registry and management
- `pkg/tools/` - Tool implementations (SpawnAgent, SendMessage, RemoveAgent, etc.)
- `pkg/agents/` - Agent implementations and providers
- `pkg/mocks/` - Centralized mock files for testing
- `pkg/di/` - Dependency injection container

## Agent Tools

### Available Tools
- **SpawnAgentTool**: Creates new subagents with custom system prompts
- **DEPRECATED SendMessageTool**: Sends messages between agents
- **RemoveAgentTool**: Removes agents and all their subagents recursively
- **CurrentTimeTool**: Provides current time information

### Tool Implementation Pattern
1. Create tool struct with registry and senderID
2. Implement provider interface for DI
3. Add to DI container in `pkg/di/container.go`
4. Register in default agent configuration in `pkg/agents/default.go`

### Testing Tools
- Use centralized mocks from `pkg/mocks/`
- Follow GoMock patterns with `gomock.Controller`
- Test both success and error cases
- Validate permissions and edge cases
