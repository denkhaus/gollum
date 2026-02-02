// Package mocks provides generated mock implementations for testing.
// Use go generate to create mock files from source interfaces.
package mocks

//go:generate go run go.uber.org/mock/mockgen -source=../registry/registry.go -destination=mock_registry.go -package=mocks github.com/denkhaus/gollum/pkg/registry AgentRegistry
//go:generate go run go.uber.org/mock/mockgen -source=../config/service.go -destination=mock_config_service.go -package=mocks github.com/denkhaus/gollum/pkg/config ConfigService
//go:generate go run go.uber.org/mock/mockgen -source=../shared/agent.go -destination=mock_agent.go -package=mocks github.com/denkhaus/gollum/pkg/shared Agent
//go:generate go run go.uber.org/mock/mockgen -source=../shared/factory.go -destination=mock_agent_factory.go -package=mocks github.com/denkhaus/gollum/pkg/shared AgentFactory
//go:generate go run go.uber.org/mock/mockgen -source=../state/types.go -destination=mock_file_state_manager.go -package=mocks github.com/denkhaus/gollum/pkg/state FileStateManager
//go:generate go run go.uber.org/mock/mockgen -source=../logger/logger.go -destination=mock_logger_service.go -package=mocks github.com/denkhaus/gollum/pkg/logger LoggerService
//go:generate go run go.uber.org/mock/mockgen -source=../prompt/manager/manager.go -destination=mock_prompt_manager.go -package=mocks github.com/denkhaus/gollum/pkg/prompt/manager PromptManager
//go:generate go run go.uber.org/mock/mockgen -source=../prompt/store/store.go -destination=mock_store.go -package=mocks github.com/denkhaus/gollum/pkg/prompt/store PromptStore
//go:generate go run go.uber.org/mock/mockgen -source=../prompt/optimizer/optimizer.go -destination=mock_optimizer.go -package=mocks github.com/denkhaus/gollum/pkg/prompt/optimizer PromptOptimizer
//go:generate go run go.uber.org/mock/mockgen -source=../llm/provider.go -destination=mock_llm_client_provider.go -package=mocks github.com/denkhaus/gollum/pkg/llm ClientProvider
//go:generate go run go.uber.org/mock/mockgen -source=../ui/agent_messenger.go -destination=mock_agent_messenger.go -package=mocks github.com/denkhaus/gollum/pkg/ui AgentMessenger
//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_llm_client.go -package=mocks github.com/m-mizutani/gollem LLMClient
//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_session.go -package=mocks github.com/m-mizutani/gollem Session
//go:generate go run go.uber.org/mock/mockgen -source=../tools/agent_execution_helper.go -destination=mock_agent_execution_helper.go -package=mocks github.com/denkhaus/gollum/pkg/tools AgentExecutionHelper
//go:generate go run go.uber.org/mock/mockgen -source=../hooks/manager.go -destination=mock_hook_manager.go -package=mocks github.com/denkhaus/gollum/pkg/hooks HookManager
