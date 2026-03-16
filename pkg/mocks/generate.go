// Package mocks provides generated mock implementations for EXTERNAL dependencies only.
//
// IMPORTANT: This package is exclusively for mocking external (third-party) interfaces.
// Internal service mocks have been moved to their respective service packages to avoid import cycles.
//
// To add a mock for an external dependency:
// 1. Add a go:generate directive below with the external package path
// 2. Run: go generate ./pkg/mocks/generate.go
//
// External dependencies mocked here:
// - github.com/m-mizutani/gollem (LLMClient, Session)
package mocks

//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_llm_client.go -package=mocks github.com/m-mizutani/gollem LLMClient

//go:generate go run -tags=mock go.uber.org/mock/mockgen -destination=mock_session.go -package=mocks github.com/m-mizutani/gollem Session
