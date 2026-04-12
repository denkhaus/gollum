// pkg/channel/types_test.go
package channel

import (
	"context"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannelIdentifier_String(t *testing.T) {
	id := ChannelIdentifier("tui")
	assert.Equal(t, "tui", string(id))
}

func TestChannelFactory_CreatesChannel(t *testing.T) {
	factory := func(opts ...ChannelOption) (Channel, error) {
		return &mockChannelForTypes{id: uuid.New()}, nil
	}

	ch, err := factory()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestChannelOption_Apply(t *testing.T) {
	ch := &mockChannelForTypes{id: uuid.New()}
	opt := &mockOption{applyFunc: func(c Channel) error {
		return nil
	}}

	err := opt.Apply(ch)
	assert.NoError(t, err)
}

func TestChannel_Start_Interface(t *testing.T) {
	// Compile-time check that Channel.Start is part of the interface
	var _ Channel = (*mockStarterChannel)(nil)
}

// mockChannelForTypes for testing (avoiding conflict with facade_test.go)
type mockChannelForTypes struct {
	id uuid.UUID
}

func (m *mockChannelForTypes) ID() uuid.UUID                              { return m.id }
func (m *mockChannelForTypes) OnMessage(msg Message)                      {}
func (m *mockChannelForTypes) OnLog(entry shared.LogEntry)                {}
func (m *mockChannelForTypes) OnAgentLifecycle(event AgentLifecycleEvent) {}
func (m *mockChannelForTypes) Start(ctx context.Context) error            { return nil }

// mockOption for testing
type mockOption struct {
	applyFunc func(Channel) error
}

func (m *mockOption) Apply(ch Channel) error {
	return m.applyFunc(ch)
}

// mockStarterChannel for interface verification
type mockStarterChannel struct {
	id uuid.UUID
}

func (m *mockStarterChannel) ID() uuid.UUID                              { return m.id }
func (m *mockStarterChannel) OnMessage(msg Message)                      {}
func (m *mockStarterChannel) OnLog(entry shared.LogEntry)                {}
func (m *mockStarterChannel) OnAgentLifecycle(event AgentLifecycleEvent) {}
func (m *mockStarterChannel) Start(ctx context.Context) error            { return nil }
