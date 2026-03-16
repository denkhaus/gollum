package pkg

//go:generate go run go.uber.org/mock/mockgen -source=channel/facade.go -destination=channel/facade_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel ChannelFacadeService

//go:generate go run go.uber.org/mock/mockgen -source=config/service.go -destination=config/service_mock.go -package=config github.com/denkhaus/gollum/pkg/config Service

//go:generate go run go.uber.org/mock/mockgen -source=logger/logger.go -destination=logger/logger_mock.go -package=logger github.com/denkhaus/gollum/pkg/logger LoggerService

//go:generate go run go.uber.org/mock/mockgen -source=hooks/manager.go -destination=hooks/manager_mock.go -package=hooks github.com/denkhaus/gollum/pkg/hooks HookManager

//go:generate go run go.uber.org/mock/mockgen -source=shared/agent.go -destination=shared/agent_mock.go -package=shared github.com/denkhaus/gollum/pkg/shared Agent
