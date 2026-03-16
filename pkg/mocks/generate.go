// Package mocks provides generated mock implementations for testing.
// Use go generate to create mock files from source interfaces.
//
// NOTE: Most mocks have been moved to their respective service packages to avoid import cycles.
// Only mocks that would cause cycles when placed in their service package remain here.
package mocks

// Mocks that remain in central package (would cause import cycles in their own packages):

//go:generate go run go.uber.org/mock/mockgen -source=../tools/agent_execution_helper.go -destination=mock_agent_execution_helper.go -package=mocks github.com/denkhaus/gollum/pkg/tools AgentExecutionHelper

//go:generate go run go.uber.org/mock/mockgen -source=../tui/model.go -destination=mock_agent_executor.go -package=mocks github.com/denkhaus/gollum/pkg/tui AgentExecutor

//go:generate go run go.uber.org/mock/mockgen -source=../diff/provider.go -destination=mock_diff_provider.go -package=mocks github.com/denkhaus/gollum/pkg/diff Provider

//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_llm_client.go -package=mocks github.com/m-mizutani/gollem LLMClient

//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_session.go -package=mocks github.com/m-mizutani/gollem Session
