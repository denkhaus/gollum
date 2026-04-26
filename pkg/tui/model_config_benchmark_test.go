package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HistoryMaxSize != 1000 {
		t.Error("Default history max size should be 1000")
	}

	if !config.EnableTimestamps {
		t.Error("Default should enable timestamps")
	}

	if !config.EnableColors {
		t.Error("Default should enable colors")
	}

	if !config.StatusEnabled {
		t.Error("Default should enable status bar")
	}

	if !config.MultiLineEnabled {
		t.Error("Default should enable multi-line input")
	}
}

func TestSetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	newConfig := Config{
		HistoryMaxSize:   500,
		EnableTimestamps: false,
		EnableColors:     false,
		StatusEnabled:    false,
		MultiLineEnabled: false,
	}

	m.SetConfig(newConfig)

	if m.config.HistoryMaxSize != 500 {
		t.Error("SetConfig should update history max size")
	}

	if m.config.EnableTimestamps {
		t.Error("SetConfig should update timestamps setting")
	}
}

// Benchmark tests for performance hot paths

// BenchmarkUpdateViewportContent measures performance of viewport content rebuilding
func BenchmarkUpdateViewportContent(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Add realistic messages to simulate typical TUI workload
	m.messages = []shared.Message{
		{ID: uuid.New(), Type: shared.MessageTypeSystemInfo, Content: "System initialization message", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeAgentChat, Content: "Agent response 1", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeAgentChat, Content: "Agent response 2 with some text that needs markdown rendering", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeToolResponse, Content: "Tool execution result", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeUserChat, Content: "User input message", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeUserChat, Content: "Another user message for realistic workload", Timestamp: time.Now()},
	}

	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		m.updateViewportContent()
	}

	b.StopTimer()

	// Report result
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/op")
	b.Logf("BenchmarkUpdateViewportContent: %d messages", len(m.messages))
}

// BenchmarkFormatMessage measures performance of message formatting
func BenchmarkFormatMessage(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Test with different message types to cover all code paths
	testMessages := []shared.Message{
		{ID: uuid.New(), Type: shared.MessageTypeAgentChat, Content: "Simple agent response", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeAgentChat, Content: "Response with markdown formatting: **bold** text and `code`", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeToolResponse, Content: "Tool execution output", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeUserChat, Content: "User message with emoji 🎉", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeSystemInfo, Content: "System notification", Timestamp: time.Now()},
	}

	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		for idx, msg := range testMessages {
			_ = m.formatMessage(idx, msg)
		}
	}

	b.StopTimer()

	// Report metrics
	b.ReportAllocs()
	b.Logf("BenchmarkFormatMessage: %d messages", len(testMessages))
}

// BenchmarkModelUpdate measures performance of Model.Update with message append
func BenchmarkModelUpdate(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	b.ResetTimer()

	// Simulate realistic workload: multiple messages being appended
	baseMessageCount := 100
	for i := 0; i < baseMessageCount; i++ {
		msg := shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeAgentChat,
			Content:   fmt.Sprintf("Agent response message %d", i),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
	}

	// Measure the update operation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
	}

	b.StopTimer()

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/op")
	b.Logf("BenchmarkModelUpdate: %d messages", baseMessageCount)
}

func BenchmarkRenderStatusBar(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.config.StatusEnabled = true
	statusBar := m.renderStatusBar()
	if statusBar == "" {
		b.Fatal("renderStatusBar should not return empty string")
	}
	if !strings.Contains(statusBar, "Messages: 0") {
		b.Fatal("Status bar should show message count")
	}
	if !strings.Contains(statusBar, "Idle") {
		b.Fatal("Status bar should show idle status")
	}
}

// BenchmarkUpdateViewportContentWithScrolling tests realistic TUI scrolling behavior
// In real usage: 100+ messages in history, viewport shows ~20-30 messages
// User scrolls → updateViewportContent() called repeatedly (hot path!)
func BenchmarkUpdateViewportContentWithScrolling(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Realistic TUI session: 100 messages in history
	m.messages = make([]shared.Message, 100)
	for i := range m.messages {
		m.messages[i] = shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeAgentChat,
			Content:   fmt.Sprintf("Agent response message %d", i),
			Timestamp: time.Now(),
		}
	}

	// Initialize viewport once
	m.viewport.SetContent(m.updateViewportContent())

	b.ResetTimer()

	// Simulate realistic scrolling: updateViewportContent called multiple times per "scroll" event
	// This measures the hot path cost that occurs during actual TUI usage
	for i := 0; i < b.N; i++ {
		m.updateViewportContent()
	}

	b.StopTimer()

	// Report metrics
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/op")
	b.Logf("BenchmarkUpdateViewportContentWithScrolling: 100 messages, %d scroll updates", len(m.messages))
}

// BenchmarkFormatMessageWithCache tests cache hit performance.
// This benchmark measures the performance improvement from caching.
func BenchmarkFormatMessageWithCache(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create test messages
	testMessages := make([]shared.Message, 10)
	for i := range testMessages {
		testMessages[i] = shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeAgentChat,
			Content:   fmt.Sprintf("Agent response message %d with some markdown formatting **bold** and `code`", i),
			Timestamp: time.Now(),
		}
	}

	// First pass: warm up the cache (cache miss)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for idx, msg := range testMessages {
			_ = m.formatMessage(idx, msg)
		}
	}

	b.StopTimer()

	// Report metrics showing cache effectiveness
	b.ReportAllocs()
	b.Logf("BenchmarkFormatMessageWithCache: %d messages, %d iterations, cache warmed", len(testMessages), b.N)
}

// BenchmarkFormatMessageNoCache tests cache miss performance.
// This benchmark measures formatting cost when cache is cold.
func BenchmarkFormatMessageNoCache(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create test messages with unique IDs (always cache miss)
	testMessages := make([]shared.Message, 10)

	b.ResetTimer()

	// Force cache miss by clearing cache each iteration
	for i := 0; i < b.N; i++ {
		// Clear cache to simulate cold start
		m.clearFormatCache()

		// Create new messages for each iteration (UUID will be different)
		for j := range testMessages {
			testMessages[j] = shared.Message{
				ID:        uuid.New(),
				Type:      shared.MessageTypeAgentChat,
				Content:   fmt.Sprintf("Agent response message %d with some markdown formatting **bold** and `code`", j),
				Timestamp: time.Now(),
			}
			_ = m.formatMessage(j, testMessages[j])
		}
	}

	b.StopTimer()

	// Report metrics
	b.ReportAllocs()
	b.Logf("BenchmarkFormatMessageNoCache: %d messages, %d iterations, cache cleared each time", len(testMessages), b.N)
}
