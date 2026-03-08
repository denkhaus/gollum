package extensions

import (
	"github.com/samber/do/v2"
)

// DIGateway exposes the DI injector to extensions for service registration and consumption.
// This is a minimal interface that can be extended non-breakingly in the future.
type DIGateway interface {
	// Injector returns the raw DI injector for extensions to use.
	// Extensions can use do.Provide() to register services
	// and do.MustInvoke[T]() to consume services.
	Injector() do.Injector
}

// gatewayServiceImpl is the private implementation
type gatewayServiceImpl struct {
	injector do.Injector
}

// Ensure gatewayServiceImpl implements DIGateway at compile time
var _ DIGateway = (*gatewayServiceImpl)(nil)

// NewGatewayService creates the DI gateway service.
// This is typically called during DI container initialization.
func NewGatewayService(injector do.Injector) (DIGateway, error) {
	if injector == nil {
		return nil, ErrInvalidInjector
	}
	return &gatewayServiceImpl{
		injector: injector,
	}, nil
}

func (p *gatewayServiceImpl) Injector() do.Injector {
	return p.injector
}
