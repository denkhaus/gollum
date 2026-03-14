// Package mocks provides generated mock implementations for testing in pkg/app.
// Generated mocks are placed here to avoid import cycles.
package mocks

//go:generate go run go.uber.org/mock/mockgen -source=../../flows/executor/executor.go -destination=mock_flow_executor.go -package=mocks github.com/denkhaus/gollum/pkg/flows/executor FlowExecutorService,FlowExecutorInstance
//go:generate go run go.uber.org/mock/mockgen -source=../../channel/facade.go -destination=mock_channel_facade.go -package=mocks github.com/denkhaus/gollum/pkg/channel ChannelFacadeService
