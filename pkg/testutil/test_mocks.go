package testutil

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/session"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// MocksWithController holds common mock instances for testing
type MocksWithController struct {
	Ctrl        *gomock.Controller
	Logger      *logger.MockLoggerService
	HookManager *hooks.MockHookManager
	SessionMgr  *session.MockSessionManager
}

// NewCommonMocks creates a set of commonly used mocks
// This reduces boilerplate in test setup
func NewCommonMocks(t testing.TB) *MocksWithController {
	ctrl := gomock.NewController(t)

	return &MocksWithController{
		Ctrl:        ctrl,
		Logger:      logger.NewMockLoggerService(ctrl),
		HookManager: hooks.NewMockHookManager(ctrl),
		SessionMgr:  session.NewMockSessionManager(ctrl),
	}
}

// SetupPassThrough configures all mocks to pass through calls
func (m *MocksWithController) SetupPassThrough() {
	SetupMockLoggerPassThrough(m.Logger)
	m.setupHookManagerPassThrough()
	m.setupSessionManagerPassThrough()
}

func (m *MocksWithController) setupHookManagerPassThrough() {
	m.HookManager.EXPECT().WithToolHooks(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).DoAndReturn(func(_ interface{}, _ interface{}, _ string, _ map[string]any, workFunc func() (map[string]any, error)) (map[string]any, error) {
		return workFunc()
	}).AnyTimes()
}

func (m *MocksWithController) setupSessionManagerPassThrough() {
	testUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sctx := &shared.SessionContext{
		SessionID: testUUID,
		ChannelID: testUUID,
	}
	m.SessionMgr.EXPECT().CreateSession(gomock.Any(), gomock.Any()).
		Return(&shared.Session{
			SessionContext: *sctx,
		}, nil).AnyTimes()
	m.SessionMgr.EXPECT().GetSession(gomock.Any()).
		Return(&shared.Session{
			SessionContext: *sctx,
		}, true).AnyTimes()
	m.SessionMgr.EXPECT().GetOrCreateSession(gomock.Any(), gomock.Any()).
		Return(&shared.Session{
			SessionContext: *sctx,
		}, nil).AnyTimes()
	m.SessionMgr.EXPECT().GetSessionsByChannel(gomock.Any()).
		Return([]*shared.Session{}).AnyTimes()
	m.SessionMgr.EXPECT().CloseSession(gomock.Any()).
		Return(nil).AnyTimes()
}
