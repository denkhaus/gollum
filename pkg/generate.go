package pkg

//go:generate go run go.uber.org/mock/mockgen -source=channel/facade.go -destination=channel/facade_mock.go -package=channel github.com/denkhaus/gollum/pkg/channel ChannelFacadeService

//go:generate go run go.uber.org/mock/mockgen -source=config/service.go -destination=config/service_mock.go -package=config github.com/denkhaus/gollum/pkg/config Service

//go:generate go run go.uber.org/mock/mockgen -source=logger/logger.go -destination=logger/logger_mock.go -package=logger github.com/denkhaus/gollum/pkg/logger LoggerService

//go:generate go run go.uber.org/mock/mockgen -source=hooks/manager.go -destination=hooks/manager_mock.go -package=hooks github.com/denkhaus/gollum/pkg/hooks HookManager

//go:generate go run go.uber.org/mock/mockgen -source=shared/agent.go -destination=shared/agent_mock.go -package=shared github.com/denkhaus/gollum/pkg/shared Agent

//go:generate go run go.uber.org/mock/mockgen -source=registry/registry.go -destination=registry/registry_mock.go -package=registry github.com/denkhaus/gollum/pkg/registry AgentRegistry

//go:generate go run go.uber.org/mock/mockgen -source=shared/factory.go -destination=shared/factory_mock.go -package=shared github.com/denkhaus/gollum/pkg/shared AgentFactory

//go:generate go run go.uber.org/mock/mockgen -source=prompt/manager/manager.go -destination=prompt/manager/manager_mock.go -package=manager github.com/denkhaus/gollum/pkg/prompt/manager PromptManager

//go:generate go run go.uber.org/mock/mockgen -source=state/types.go -destination=state/types_mock.go -package=state github.com/denkhaus/gollum/pkg/state FileStateManager

//go:generate go run go.uber.org/mock/mockgen -source=skills/service.go -destination=skills/service_mock.go -package=skills github.com/denkhaus/gollum/pkg/skills SkillService

//go:generate go run go.uber.org/mock/mockgen -source=workspace/service.go -destination=workspace/service_mock.go -package=workspace github.com/denkhaus/gollum/pkg/workspace Service

//go:generate go run go.uber.org/mock/mockgen -source=events/bus.go -destination=events/bus_mock.go -package=events github.com/denkhaus/gollum/pkg/events Bus

//go:generate go run go.uber.org/mock/mockgen -source=markdown/renderer.go -destination=markdown/renderer_mock.go -package=markdown github.com/denkhaus/gollum/pkg/markdown Renderer
