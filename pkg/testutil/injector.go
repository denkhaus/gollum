package testutil

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// TestInjectorConfig holds configuration for test injector setup
type TestInjectorConfig struct {
	AgentLimits *config.AgentLimitsConfig
}

// DefaultTestInjectorConfig returns default configuration for test injector
func DefaultTestInjectorConfig() TestInjectorConfig {
	return TestInjectorConfig{
		AgentLimits: &config.AgentLimitsConfig{
			MaxSubAgentsPerParent: 3,
			MaxTotalAgents:        50,
		},
	}
}

// NewTestInjector creates a test injector with common services pre-configured.
//
// Example usage:
//
//	injector := testutil.NewTestInjector(t)
func NewTestInjector(t testing.TB, opts ...func(*TestInjectorConfig)) do.Injector {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := DefaultTestInjectorConfig()
	for _, opt := range opts {
		opt(&config)
	}

	injector := do.New()

	setupMockConfigService(injector, ctrl, config.AgentLimits)
	do.Provide(injector, logger.NewService)
	setupMockEventBus(injector, ctrl)
	setupMockPromptManager(injector, ctrl)
	setupMockWorkspaceService(injector, ctrl)
	setupMockSkillService(injector, ctrl)

	return injector
}

func setupMockConfigService(injector do.Injector, ctrl *gomock.Controller, limits *config.AgentLimitsConfig) {
	mockConfigService := config.NewMockConfigService(ctrl)
	mockConfigService.EXPECT().GetAgentLimits().Return(limits).AnyTimes()
	mockConfigService.EXPECT().IsDevMode().Return(false).AnyTimes()
	mockConfigService.EXPECT().GetLogLevel().Return("info").AnyTimes()
	mockConfigService.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
		SessionLogBufferSize: 1000,
		SessionLogEnabled:    true,
	}).AnyTimes()
	mockConfigService.EXPECT().GetEventsConfig().Return(&config.EventsConfig{
		MaxRetries:   3,
		RetryDelayMs: 100,
		RetryBackoff: 2,
	}).AnyTimes()
	mockConfigService.EXPECT().GetDatabaseConfig().Return(config.DatabaseConfig{}).AnyTimes()
	do.ProvideValue[config.ConfigService](injector, mockConfigService)
}

func setupMockEventBus(injector do.Injector, ctrl *gomock.Controller) {
	mockEventBus := events.NewMockBus(ctrl)
	mockEventBus.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("test-subscription-id", nil).AnyTimes()
	do.ProvideValue[events.Bus](injector, mockEventBus)
}

func setupMockPromptManager(injector do.Injector, ctrl *gomock.Controller) {
	mockPromptManager := manager.NewMockPromptManager(ctrl)
	do.ProvideValue[manager.PromptManager](injector, mockPromptManager)
}

func setupMockWorkspaceService(injector do.Injector, ctrl *gomock.Controller) {
	mockWorkspaceService := workspace.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").AnyTimes()
	do.ProvideValue[workspace.Service](injector, mockWorkspaceService)
}

func setupMockSkillService(injector do.Injector, ctrl *gomock.Controller) {
	mockSkillService := skills.NewMockSkillService(ctrl)
	mockSkillService.EXPECT().GetSkillsXML().Return("").AnyTimes()
	mockSkillService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).AnyTimes()
	do.ProvideValue[skills.SkillService](injector, mockSkillService)
}
