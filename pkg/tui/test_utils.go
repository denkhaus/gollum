package tui

import (
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// setupMockAgent creates a mock agent with default expectations for TUI tests.
// This helper is in a separate file (not *_test.go) so it can be shared across test files.
func setupMockAgent(ctrl *gomock.Controller) *mocks.MockAgentExecutor {
	mockAgent := mocks.NewMockAgentExecutor(ctrl)
	// Default: return simple response for any Execute call
	mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(&gollem.ExecuteResponse{Texts: []string{"response"}}, nil).
		AnyTimes()
	return mockAgent
}
