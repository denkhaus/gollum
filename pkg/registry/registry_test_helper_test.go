package registry

import (
	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/events"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/skills"
	"github.com/denkhaus/gollum/pkg/workspace"
	"github.com/samber/do/v2"
	"go.uber.org/mock/gomock"
)

// setupTestInjectorWithLimits creates an injector with all required services for testing
func setupTestInjectorWithLimits(ctrl *gomock.Controller, limits *config.AgentLimitsConfig) do.Injector {
	// Create a mock config service
	mockConfigService := mocks.NewMockConfigService(ctrl)
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

	// Create a mock event bus
	mockEventBus := mocks.NewMockBus(ctrl)
	mockEventBus.EXPECT().Subscribe(
		events.EventSkillsUpdated.String(),
		gomock.Any(),
		gomock.Any(),
	).Return("test-subscription-id", nil).AnyTimes()
	mockEventBus.EXPECT().Subscribe(
		events.EventDirectoryChanged.String(),
		gomock.Any(),
		gomock.Any(),
	).Return("test-subscription-id-2", nil).AnyTimes()

	// Create a mock prompt manager
	mockPromptManager := mocks.NewMockPromptManager(ctrl)

	// Create a mock workspace service
	mockWorkspaceService := mocks.NewMockService(ctrl)
	mockWorkspaceService.EXPECT().GetCurrentWorkspace().Return("/test/workspace").AnyTimes()

	// Create a mock skill service
	mockSkillService := mocks.NewMockSkillService(ctrl)
	mockSkillService.EXPECT().GetSkillsXML().Return("").AnyTimes()
	mockSkillService.EXPECT().GetSkillInfos().Return([]shared.SkillInfo{}).AnyTimes()

	// Create a mock injector
	injector := do.New()
	do.ProvideValue[config.ConfigService](injector, mockConfigService)
	do.Provide(injector, logger.NewService)
	do.ProvideValue[events.Bus](injector, mockEventBus)
	do.ProvideValue[manager.PromptManager](injector, mockPromptManager)
	do.ProvideValue[workspace.Service](injector, mockWorkspaceService)
	do.ProvideValue[skills.SkillService](injector, mockSkillService)

	return injector
}
