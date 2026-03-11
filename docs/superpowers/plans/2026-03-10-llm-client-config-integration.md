# LLMClientConfig Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `LLMProvider` field in `AgentConfig` with `LLMClientConfig`, supporting "provider/model" format and optional LLM parameters with fallback to config service defaults.

**Architecture:**
- Parse "provider/model" format in `LLMClientConfig` methods
- Update `ClientProvider.GetClient` to use parsed config with fallback defaults
- Replace all `LLMProvider` usages with `LLMClientConfig` across codebase

**Tech Stack:** Go 1.23+, gollem LLM library, envconfig for configuration

---

## Task 1: Add ErrInvalidModelFormat to shared package

**Files:**
- Modify: `pkg/shared/shared.go:141`

**Step 1: Add the error constant**

Add after line 141 (`ErrLLMProviderNotSupported`):

```go
ErrInvalidModelFormat = errors.New("model must be in 'provider/model' format")
```

**Step 2: Run tests to verify no existing tests break**

Run: `go test ./pkg/shared/...`
Expected: PASS (no existing tests should be affected)

**Step 3: Commit**

```bash
git add pkg/shared/shared.go
git commit -m "feat(shared): add ErrInvalidModelFormat error"
```

---

## Task 2: Add Provider() method to LLMClientConfig

**Files:**
- Modify: `pkg/shared/agent.go:41-42`
- Test: `pkg/shared/agent_test.go` (create new)

**Step 1: Write the failing test**

Create `pkg/shared/agent_test.go`:

```go
package shared

import (
	"testing"
)

func TestLLMClientConfigProvider(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		want      LLMProvider
		wantErr   error
	}{
		{
			name:    "anthropic provider",
			model:   "anthropic/claude-3-5-sonnet-20241022",
			want:    LLMProviderAnthropic,
			wantErr: nil,
		},
		{
			name:    "openai provider",
			model:   "openai/gpt-4o",
			want:    LLMProviderOpenAI,
			wantErr: nil,
		},
		{
			name:    "gemini provider",
			model:   "gemini/gemini-2.0-flash-exp",
			want:    LLMProviderGemini,
			wantErr: nil,
		},
		{
			name:    "no slash - invalid format",
			model:   "claude-3-5-sonnet-20241022",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "empty string - invalid format",
			model:   "",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "only slash - invalid format",
			model:   "/",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnf := &LLMClientConfig{Model: tt.model}
			got, err := cnf.Provider()

			if err != tt.wantErr {
				t.Errorf("Provider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Provider() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/shared -v -run TestLLMClientConfigProvider`
Expected: FAIL with "method Provider not defined"

**Step 3: Write minimal implementation**

Add to `pkg/shared/agent.go` after the `LLMClientConfig` struct:

```go
// Provider parses and returns the LLM provider from the Model field.
// Model must be in "provider/model" format (e.g., "anthropic/claude-3-5-sonnet-20241022").
// Returns ErrInvalidModelFormat if no slash is present.
func (c *LLMClientConfig) Provider() (LLMProvider, error) {
	idx := strings.Index(c.Model, "/")
	if idx == -1 || idx == 0 || idx == len(c.Model)-1 {
		return "", ErrInvalidModelFormat
	}
	providerStr := c.Model[:idx]
	switch LLMProvider(providerStr) {
	case LLMProviderAnthropic, LLMProviderOpenAI, LLMProviderGemini:
		return LLMProvider(providerStr), nil
	default:
		return "", ErrLLMProviderNotSupported
	}
}
```

Also add import at top of file:
```go
import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/shared -v -run TestLLMClientConfigProvider`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/shared/agent.go pkg/shared/agent_test.go
git commit -m "feat(shared): add Provider() method to LLMClientConfig with tests"
```

---

## Task 3: Add ModelName() method to LLMClientConfig

**Files:**
- Modify: `pkg/shared/agent.go` (after Provider method)
- Test: `pkg/shared/agent_test.go`

**Step 1: Write the failing test**

Add to `pkg/shared/agent_test.go`:

```go
func TestLLMClientConfigModelName(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		want      string
		wantErr   error
	}{
		{
			name:    "anthropic model",
			model:   "anthropic/claude-3-5-sonnet-20241022",
			want:    "claude-3-5-sonnet-20241022",
			wantErr: nil,
		},
		{
			name:    "openai model",
			model:   "openai/gpt-4o",
			want:    "gpt-4o",
			wantErr: nil,
		},
		{
			name:    "gemini model",
			model:   "gemini/gemini-2.0-flash-exp",
			want:    "gemini-2.0-flash-exp",
			wantErr: nil,
		},
		{
			name:    "no slash - invalid format",
			model:   "claude-3-5-sonnet-20241022",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "empty string - invalid format",
			model:   "",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
		{
			name:    "trailing slash",
			model:   "anthropic/",
			want:    "",
			wantErr: ErrInvalidModelFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnf := &LLMClientConfig{Model: tt.model}
			got, err := cnf.ModelName()

			if err != tt.wantErr {
				t.Errorf("ModelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ModelName() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/shared -v -run TestLLMClientConfigModelName`
Expected: FAIL with "method ModelName not defined"

**Step 3: Write minimal implementation**

Add to `pkg/shared/agent.go` after `Provider()` method:

```go
// ModelName returns just the model name portion (after the slash).
// Returns ErrInvalidModelFormat if no slash is present.
func (c *LLMClientConfig) ModelName() (string, error) {
	idx := strings.Index(c.Model, "/")
	if idx == -1 || idx == 0 || idx == len(c.Model)-1 {
		return "", ErrInvalidModelFormat
	}
	return c.Model[idx+1:], nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/shared -v -run TestLLMClientConfigModelName`
Expected: PASS

**Step 5: Run all shared package tests**

Run: `go test ./pkg/shared -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/shared/agent.go pkg/shared/agent_test.go
git commit -m "feat(shared): add ModelName() method to LLMClientConfig with tests"
```

---

## Task 4: Update ClientProvider interface and implementation

**Files:**
- Modify: `pkg/llm/provider.go:24-27` (interface)
- Modify: `pkg/llm/provider.go:43-93` (implementation)

**Step 1: Update the interface**

Change line 26 from:
```go
	GetClient(ctx context.Context, provider shared.LLMProvider) (gollem.LLMClient, error)
```

To:
```go
	GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error)
```

**Step 2: Write test for GetClient with full config**

Create `pkg/llm/provider_test.go`:

```go
package llm

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
)

// TestGetClient_AnthropicWithConfig tests GetClient with Anthropic provider and full config
func TestGetClient_AnthropicWithConfig(t *testing.T) {
	// Setup DI injector
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &config.AnthropicConfig{
			APIKey:      "test-key",
			BaseURL:     "https://api.anthropic.com",
			Model:       "claude-3-5-sonnet-20241022",
			Temperature: 0.7,
			MaxTokens:   8192,
			TopP:        1.0,
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	temp := 0.5
	maxTokens := 4096
	topP := 0.9

	cnf := &shared.LLMClientConfig{
		Model:       "anthropic/claude-3-5-sonnet-20241022",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
	}

	client, err := provider.GetClient(ctx, cnf)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Error("GetClient() returned nil client")
	}
}

// TestGetClient_AnthropicWithDefaults tests GetClient with Anthropic provider using defaults
func TestGetClient_AnthropicWithDefaults(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &config.AnthropicConfig{
			APIKey:      "test-key",
			BaseURL:     "https://api.anthropic.com",
			Model:       "claude-3-5-sonnet-20241022",
			Temperature: 0.7,
			MaxTokens:   8192,
			TopP:        1.0,
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "anthropic/claude-3-opus-20250219",
		// Temperature, MaxTokens, TopP are nil - should use defaults
	}

	client, err := provider.GetClient(ctx, cnf)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Error("GetClient() returned nil client")
	}
}

// TestGetClient_InvalidModelFormat tests GetClient with invalid model format
func TestGetClient_InvalidModelFormat(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &config.AnthropicConfig{
			APIKey: "test-key",
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "invalid-model-format",
	}

	_, err = provider.GetClient(ctx, cnf)
	if err != shared.ErrInvalidModelFormat {
		t.Errorf("GetClient() error = %v, want %v", err, shared.ErrInvalidModelFormat)
	}
}

// TestGetClient_MissingAPIKey tests GetClient when provider API key is not configured
func TestGetClient_MissingAPIKey(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (config.ConfigService, error) {
		return &config.AnthropicConfig{
			APIKey: "", // Empty API key
		}, nil
	})

	provider, err := NewClientProvider(injector)
	if err != nil {
		t.Fatalf("NewClientProvider() error = %v", err)
	}

	ctx := context.Background()
	cnf := &shared.LLMClientConfig{
		Model: "anthropic/claude-3-5-sonnet-20241022",
	}

	_, err = provider.GetClient(ctx, cnf)
	if err == nil {
		t.Error("GetClient() expected error for missing API key, got nil")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./pkg/llm -v -run TestGetClient`
Expected: FAIL (signature mismatch, implementation still uses old `provider` parameter)

**Step 4: Update GetClient implementation**

Replace the entire `GetClient` method (lines 43-93) with:

```go
func (p *clientProvider) GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error) {
	// Parse provider and model from cnf.Model
	provider, err := cnf.Provider()
	if err != nil {
		return nil, err
	}

	modelName, err := cnf.ModelName()
	if err != nil {
		return nil, err
	}

	switch provider {
	case shared.LLMProviderGemini:
		cfg := p.configService.GetGeminiConfig()
		if cfg.ProjectID == "" {
			return nil, fmt.Errorf("gemini provider not configured: missing project_id")
		}

		// Use cnf values if provided, otherwise use defaults
		temp := cfg.Temperature
		if cnf.Temperature != nil {
			temp = *cnf.Temperature
		}
		maxTokens := cfg.MaxTokens
		if cnf.MaxTokens != nil {
			maxTokens = *cnf.MaxTokens
		}
		topP := cfg.TopP
		if cnf.TopP != nil {
			topP = *cnf.TopP
		}

		client, err := gemini.New(ctx, cfg.ProjectID, cfg.Location,
			gemini.WithModel(modelName),
			gemini.WithTemperature(float32(temp)),
			gemini.WithMaxTokens(int32(maxTokens)),
			gemini.WithTopP(float32(topP)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %v", err)
		}
		return client, nil

	case shared.LLMProviderAnthropic:
		cfg := p.configService.GetAnthropicConfig()
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic provider not configured: missing API key")
		}

		temp := cfg.Temperature
		if cnf.Temperature != nil {
			temp = *cnf.Temperature
		}
		maxTokens := cfg.MaxTokens
		if cnf.MaxTokens != nil {
			maxTokens = *cnf.MaxTokens
		}
		topP := cfg.TopP
		if cnf.TopP != nil {
			topP = *cnf.TopP
		}

		client, err := claude.New(ctx, cfg.APIKey,
			claude.WithModel(modelName),
			claude.WithTemperature(temp),
			claude.WithMaxTokens(int64(maxTokens)),
			claude.WithTopP(topP),
			claude.WithBaseURL(cfg.BaseURL),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create anthropic client: %v", err)
		}
		return client, nil

	case shared.LLMProviderOpenAI:
		cfg := p.configService.GetOpenAIConfig()
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai provider not configured: missing API key")
		}

		temp := cfg.Temperature
		if cnf.Temperature != nil {
			temp = *cnf.Temperature
		}
		maxTokens := cfg.MaxTokens
		if cnf.MaxTokens != nil {
			maxTokens = *cnf.MaxTokens
		}
		topP := cfg.TopP
		if cnf.TopP != nil {
			topP = *cnf.TopP
		}

		client, err := openai.New(ctx, cfg.APIKey,
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithModel(modelName),
			openai.WithMaxTokens(maxTokens),
			openai.WithTemperature(float32(temp)),
			openai.WithTopP(float32(topP)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create openai client: %v", err)
		}
		return client, nil

	default:
		return nil, shared.ErrLLMProviderNotSupported
	}
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./pkg/llm -v -run TestGetClient`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/llm/provider.go pkg/llm/provider_test.go
git commit -m "feat(llm): update GetClient to use LLMClientConfig with fallback defaults"
```

---

## Task 5: Replace LLMProvider field with LLMClientConfig in AgentConfig

**Files:**
- Modify: `pkg/shared/shared.go:146-160`

**Step 1: Update AgentConfig struct**

Replace the `LLMProvider` field with `LLMClientConfig`:

Change line 156 from:
```go
	LLMProvider     LLMProvider      `json:"llm_provider"`
```

To:
```go
	LLMClientConfig *LLMClientConfig `json:"llm_client_config"`
```

**Step 2: Verify the struct looks correct**

Run: `go build ./pkg/shared/...`
Expected: PASS (compiles successfully)

**Step 3: Commit**

```bash
git add pkg/shared/shared.go
git commit -m "refactor(shared): replace LLMProvider with LLMClientConfig in AgentConfig"
```

---

## Task 6: Update AgentFactory.CreateAgent to use LLMClientConfig

**Files:**
- Modify: `pkg/agents/factory.go:144-150`

**Step 1: Update GetClient call**

Change lines 144-150 from:
```go
	// Get LLM client
	client, err := f.clientProvider.GetClient(ctx, config.LLMProvider)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
			WithContext("agent_id", config.ID).
			WithContext("llm_provider", config.LLMProvider)
	}
```

To:
```go
	// Get LLM client
	client, err := f.clientProvider.GetClient(ctx, config.LLMClientConfig)
	if err != nil {
		return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
			WithContext("agent_id", config.ID)
	}
```

**Step 2: Verify compilation**

Run: `go build ./pkg/agents/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/agents/factory.go
git commit -m "refactor(agents): update CreateAgent to use LLMClientConfig"
```

---

## Task 7: Update app/service.go supervisor agent creation

**Files:**
- Modify: `pkg/app/service.go:152-159`

**Step 1: Update supervisor agent config creation**

Change lines 152-159 from:
```go
	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		Role:            "Supervisor Agent",
		LLMProvider:     shared.LLMProviderAnthropic,
		ToolSets:        toolSet,
	}
```

To:
```go
	// Create agent config
	agentConfig := &shared.AgentConfig{
		AllowCompaction: true,
		SystemPrompt:    systemPrompt,
		Role:            "Supervisor Agent",
		LLMClientConfig: &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
		ToolSets: toolSet,
	}
```

**Step 2: Verify compilation**

Run: `go build ./pkg/app/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/app/service.go
git commit -m "refactor(app): update supervisor agent to use LLMClientConfig"
```

---

## Task 8: Update spawn_agent.go subagent creation

**Files:**
- Modify: `pkg/tools/spawn_agent.go:170-202`

**Step 1: Remove inferLLMProvider usage and update config**

Change lines 170-202 from:
```go
	// Get parent agent to inherit LLM provider and optionally message history
	parentAgent, hasParent := t.registry.GetAgent(t.senderID)
	var llmProvider shared.LLMProvider
	if hasParent {
		llmProvider = parentAgent.GetConfig().LLMProvider
		t.logService.Debugf("Inheriting LLM provider from parent agent %s", t.senderID)
	} else {
		llmProvider = shared.LLMProviderAnthropic // Default fallback
		t.logService.Debugf("Using default LLM provider (no parent agent found)")
	}

	var history *gollem.History
	if shareContext && hasParent {
		history, err = parentAgent.GetMessageHistory(ctx)
		if err != nil {
			t.logService.Warnf("Failed to get message history from parent agent: %v", err)
			// Continue without history - non-fatal error
		}
	}

	// Create subagent configuration
	taskID := uuid.New()
	subagentConfig := &shared.AgentConfig{
		AllowCompaction: false, // Don't allow compaction in Sub-agents
		ID:              taskID,
		ParentID:        &t.senderID,
		SystemPrompt:    systemPrompt,
		Role:            role,
		Description:     description,
		LLMProvider:     llmProvider,
		OutputMode:      shared.OutputModeSummary, // Sub-agents use summary mode
		History:         history,                  // Include parent message history for context awareness
	}
```

To:
```go
	// Get parent agent to inherit LLM client config and optionally message history
	parentAgent, hasParent := t.registry.GetAgent(t.senderID)
	var llmClientConfig *shared.LLMClientConfig
	if hasParent {
		llmClientConfig = parentAgent.GetConfig().LLMClientConfig
		t.logService.Debugf("Inheriting LLM config from parent agent %s", t.senderID)
	} else {
		// Default fallback
		llmClientConfig = &shared.LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		}
		t.logService.Debugf("Using default LLM config (no parent agent found)")
	}

	var history *gollem.History
	if shareContext && hasParent {
		history, err = parentAgent.GetMessageHistory(ctx)
		if err != nil {
			t.logService.Warnf("Failed to get message history from parent agent: %v", err)
			// Continue without history - non-fatal error
		}
	}

	// Create subagent configuration
	taskID := uuid.New()
	subagentConfig := &shared.AgentConfig{
		AllowCompaction: false, // Don't allow compaction in Sub-agents
		ID:              taskID,
		ParentID:        &t.senderID,
		SystemPrompt:    systemPrompt,
		Role:            role,
		Description:     description,
		LLMClientConfig: llmClientConfig,
		OutputMode:      shared.OutputModeSummary, // Sub-agents use summary mode
		History:         history,                  // Include parent message history for context awareness
	}
```

**Step 2: Verify compilation**

Run: `go build ./pkg/tools/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/tools/spawn_agent.go
git commit -m "refactor(tools): update spawn_agent to use LLMClientConfig"
```

---

## Task 9: Update invoke_skill.go agent creation

**Files:**
- Modify: `pkg/tools/invoke_skill.go:220-240`

**Step 1: Find and update the agent config creation**

First, read the file to find the exact location of LLMProvider usage around line 230:

```bash
grep -n "LLMProvider" pkg/tools/invoke_skill.go
```

Then update the config assignment to use `LLMClientConfig` instead of `LLMProvider`.

The pattern will be similar to spawn_agent.go - inherit from parent or use default.

**Step 2: Verify compilation**

Run: `go build ./pkg/tools/...`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/tools/invoke_skill.go
git commit -m "refactor(tools): update invoke_skill to use LLMClientConfig"
```

---

## Task 10: Update flows/executor/llm_step.go

**Files:**
- Modify: `pkg/flows/executor/llm_step.go:30-44`
- Modify: `pkg/flows/executor/llm_step.go:89-117` (remove inferLLMProvider)

**Step 1: Update agent config creation to use LLMClientConfig**

Change lines 30-44 from:
```go
	// Map flow Agent to shared.AgentConfig
	config := &shared.AgentConfig{
		ID:           uuid.New(),
		SystemPrompt: agentConfig.Prompt,
		Role:         "flow-llm-step",
		Description:  fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMProvider:  p.inferLLMProvider(agentConfig.Model),
		OutputMode:   shared.OutputModeSilent, // Suppress output during flow execution
		Strategy:     simple.New(),
		Tools:        nil, // Tools can be added later if needed
		ToolSets:     nil,
	}
```

To:
```go
	// Map flow Agent to shared.AgentConfig
	// Use flows.Agent.ToClientConfig() to convert to shared.LLMClientConfig
	config := &shared.AgentConfig{
		ID:              uuid.New(),
		SystemPrompt:    agentConfig.Prompt,
		Role:            "flow-llm-step",
		Description:     fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
		LLMClientConfig: agentConfig.ToClientConfig(),
		OutputMode:      shared.OutputModeSilent, // Suppress output during flow execution
		Strategy:        simple.New(),
		Tools:           nil, // Tools can be added later if needed
		ToolSets:        nil,
	}
```

**Step 2: Remove inferLLMProvider function**

Delete lines 89-117 (the `inferLLMProvider` function).

**Step 3: Verify compilation**

Run: `go build ./pkg/flows/executor/...`
Expected: PASS

**Step 4: Update flows/types.go ToClientConfig to add provider prefix**

Update `pkg/flows/types.go` the `ToClientConfig()` method to add the provider prefix:

Change lines 194-212 from:
```go
func (p Agent) ToClientConfig() *shared.LLMClientConfig {
	cnf := &shared.LLMClientConfig{
		Model: p.Model,
	}

	if p.MaxTokens != 0 {
		cnf.MaxTokens = &p.MaxTokens
	}

	if p.Temperature != 0.0 {
		cnf.Temperature = &p.Temperature
	}

	if p.TopP != 0.0 {
		cnf.TopP = &p.TopP
	}

	return cnf
}
```

To:
```go
func (p Agent) ToClientConfig() *shared.LLMClientConfig {
	// Infer provider from model name for backward compatibility
	provider := p.inferProvider()
	cnf := &shared.LLMClientConfig{
		Model: provider + "/" + p.Model,
	}

	if p.MaxTokens != 0 {
		cnf.MaxTokens = &p.MaxTokens
	}

	if p.Temperature != 0.0 {
		cnf.Temperature = &p.Temperature
	}

	if p.TopP != 0.0 {
		cnf.TopP = &p.TopP
	}

	return cnf
}

// inferProvider infers the LLM provider from the model name
// Default to Anthropic if unknown
func (p Agent) inferProvider() string {
	modelLower := strings.ToLower(p.Model)

	// OpenAI models
	if strings.HasPrefix(modelLower, "gpt-") ||
		strings.HasPrefix(modelLower, "o1-") ||
		strings.Contains(modelLower, "openai") {
		return "openai"
	}

	// Gemini models
	if strings.HasPrefix(modelLower, "gemini-") ||
		strings.Contains(modelLower, "google") {
		return "gemini"
	}

	// Claude models (default)
	return "anthropic"
}
```

Also add import to `pkg/flows/types.go`:
```go
import (
	"encoding/xml"
	"strings"

	"github.com/denkhaus/gollum/pkg/shared"
)
```

**Step 5: Verify compilation**

Run: `go build ./pkg/flows/...`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/flows/executor/llm_step.go pkg/flows/types.go
git commit -m "refactor(flows): update executor to use LLMClientConfig"
```

---

## Task 11: Update all test files

**Files:** (multiple test files)

**Step 1: Update shared/shared_test.go**

Change line 37 from:
```go
		LLMProvider:  LLMProviderAnthropic,
```

To:
```go
		LLMClientConfig: &LLMClientConfig{
			Model: "anthropic/claude-3-5-sonnet-20241022",
		},
```

**Step 2: Run test to verify**

Run: `go test ./pkg/shared -v`
Expected: PASS

**Step 3: Update registry tests**

Run this command to update all LLMProvider usages in registry tests:

```bash
sed -i 's/LLMProvider: shared.LLMProviderAnthropic,/LLMClientConfig: \&shared.LLMClientConfig{Model: "anthropic\/claude-3-5-sonnet-20241022"},/g' pkg/registry/*_test.go
sed -i 's/LLMProvider: "test",/LLMClientConfig: \&shared.LLMClientConfig{Model: "anthropic\/claude-3-5-sonnet-20241022"},/g' pkg/tools/*_test.go
```

**Step 4: Update tools tests**

For tools tests, also update OpenAI provider references:

```bash
sed -i 's/LLMProvider: shared.LLMProviderOpenAI,/LLMClientConfig: \&shared.LLMClientConfig{Model: "openai\/gpt-4o"},/g' pkg/tools/*_test.go
```

**Step 5: Run all tests to find remaining issues**

Run: `go test ./pkg/...`
Expected: Some tests may still need manual fixes

**Step 6: Fix any remaining test issues manually**

Check for compilation errors and fix any remaining LLMProvider usages.

**Step 7: Run full test suite**

Run: `go test ./pkg/... -v`
Expected: PASS

**Step 8: Commit**

```bash
git add pkg/
git commit -m "test: update all tests to use LLMClientConfig"
```

---

## Task 12: Update mocks

**Files:**
- Modify: `pkg/mocks/generate.go`

**Step 1: Verify mock generation source**

The mockgen source points to `pkg/shared/factory.go` which only has the AgentFactory interface.

Check if mocks need regeneration:

```bash
cd pkg/mocks && go generate
```

**Step 2: Verify mock usage in tests**

Run tests to ensure mocks still work:

Run: `go test ./pkg/...`
Expected: PASS

**Step 3: Commit if needed**

```bash
git add pkg/mocks/
git commit -m "refactor(mocks): regenerate mocks for updated interfaces"
```

---

## Task 13: Final verification

**Files:** (all)

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: PASS

**Step 2: Build project**

Run: `go build ./...`
Expected: PASS

**Step 3: Verify no LLMProvider references remain (except in constants and shared.go definitions)**

Run: `grep -r "LLMProvider:" pkg/ --include="*.go" | grep -v "test.go" | grep -v "shared.go"`
Expected: Empty output (no production code using LLMProvider field)

**Step 4: Final commit**

```bash
git add .
git commit -m "refactor: complete LLMClientConfig integration"
```

---

## Summary

This implementation plan:
1. ✅ Adds parsing methods to `LLMClientConfig` with full test coverage
2. ✅ Updates `ClientProvider.GetClient` to use `LLMClientConfig` with fallback defaults
3. ✅ Replaces `LLMProvider` field in `AgentConfig` with `LLMClientConfig`
4. ✅ Updates all production code to use the new format
5. ✅ Updates all tests to use the new format

Total estimated commits: ~13
Total estimated time: 1-2 hours
