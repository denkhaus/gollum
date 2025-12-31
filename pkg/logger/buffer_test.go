// Package logger provides unit tests for the log buffer.
package logger

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewLogBuffer tests buffer creation and initialization.
func TestNewLogBuffer(t *testing.T) {
	t.Run("Default size", func(t *testing.T) {
		buf := newLogBuffer(0, true)
		assert.Equal(t, 1000, buf.maxSize)
		assert.True(t, buf.enabled)
		assert.Equal(t, 0, buf.count)
	})

	t.Run("Custom size", func(t *testing.T) {
		buf := newLogBuffer(500, true)
		assert.Equal(t, 500, buf.maxSize)
	})

	t.Run("Disabled buffer", func(t *testing.T) {
		buf := newLogBuffer(100, false)
		assert.False(t, buf.enabled)
	})
}

// TestLogBuffer_AddAndGet tests adding entries and retrieving them.
func TestLogBuffer_AddAndGet(t *testing.T) {
	buf := newLogBuffer(5, true)

	// Add 3 entries
	entries := []LogEntry{
		{Timestamp: time.Now(), Level: "info", Message: "msg1"},
		{Timestamp: time.Now(), Level: "error", Message: "msg2"},
		{Timestamp: time.Now(), Level: "debug", Message: "msg3"},
	}

	for _, entry := range entries {
		buf.add(entry)
	}

	// Should have 3 entries
	assert.Equal(t, 3, buf.count)

	// Retrieve all entries
	result := buf.getEntries(LogFilter{})
	assert.Len(t, result, 3)

	// Check order (chronological)
	assert.Equal(t, "msg1", result[0].Message)
	assert.Equal(t, "msg2", result[1].Message)
	assert.Equal(t, "msg3", result[2].Message)
}

// TestLogBuffer_CircularOverflow tests that buffer overwrites oldest entries when full.
func TestLogBuffer_CircularOverflow(t *testing.T) {
	buf := newLogBuffer(3, true)

	// Add 5 entries (more than buffer size)
	for i := 0; i < 5; i++ {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   string(rune('a' + i)),
		})
	}

	// Buffer should be full with count = max size
	assert.Equal(t, 3, buf.count)

	// Retrieve entries - should have last 3
	result := buf.getEntries(LogFilter{})
	assert.Len(t, result, 3)

	// Should have 'c', 'd', 'e' (oldest 'a' and 'b' were overwritten)
	assert.Equal(t, "c", result[0].Message)
	assert.Equal(t, "d", result[1].Message)
	assert.Equal(t, "e", result[2].Message)
}

// TestLogBuffer_Disabled tests that disabled buffer doesn't store entries.
func TestLogBuffer_Disabled(t *testing.T) {
	buf := newLogBuffer(10, false)

	buf.add(LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "test",
	})

	// Count should still be 0
	assert.Equal(t, 0, buf.count)

	// GetEntries should return empty
	result := buf.getEntries(LogFilter{})
	assert.Empty(t, result)
}

// TestLogBuffer_FilterByLevel tests filtering by log level.
func TestLogBuffer_FilterByLevel(t *testing.T) {
	buf := newLogBuffer(10, true)

	// Add entries with different levels
	levels := []string{"info", "error", "debug", "info", "warn", "error"}
	for _, level := range levels {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   level + " message",
		})
	}

	// Filter by "info"
	result := buf.getEntries(LogFilter{Level: "info"})
	assert.Len(t, result, 2)
	for _, entry := range result {
		assert.Equal(t, "info", entry.Level)
	}

	// Filter by "error"
	result = buf.getEntries(LogFilter{Level: "error"})
	assert.Len(t, result, 2)
	for _, entry := range result {
		assert.Equal(t, "error", entry.Level)
	}

	// Filter by non-existent level
	result = buf.getEntries(LogFilter{Level: "critical"})
	assert.Empty(t, result)
}

// TestLogBuffer_FilterByAgentID tests filtering by agent ID.
func TestLogBuffer_FilterByAgentID(t *testing.T) {
	buf := newLogBuffer(10, true)

	agent1 := uuid.New()
	agent2 := uuid.New()

	// Add entries with different agents
	buf.add(LogEntry{Timestamp: time.Now(), Level: "info", Message: "msg1", AgentID: agent1})
	buf.add(LogEntry{Timestamp: time.Now(), Level: "info", Message: "msg2", AgentID: agent2})
	buf.add(LogEntry{Timestamp: time.Now(), Level: "info", Message: "msg3", AgentID: agent1})
	buf.add(LogEntry{Timestamp: time.Now(), Level: "info", Message: "msg4"}) // No agent

	// Filter by agent1
	result := buf.getEntries(LogFilter{AgentID: agent1})
	assert.Len(t, result, 2)
	for _, entry := range result {
		assert.Equal(t, agent1, entry.AgentID)
	}

	// Filter by agent2
	result = buf.getEntries(LogFilter{AgentID: agent2})
	assert.Len(t, result, 1)

	// Filter by non-existent agent
	result = buf.getEntries(LogFilter{AgentID: uuid.New()})
	assert.Empty(t, result)
}

// TestLogBuffer_FilterBySince tests filtering by timestamp.
func TestLogBuffer_FilterBySince(t *testing.T) {
	buf := newLogBuffer(10, true)

	now := time.Now()

	// Add entries at different times
	oldEntry := LogEntry{Timestamp: now.Add(-1 * time.Hour), Level: "info", Message: "old"}
	recentEntry := LogEntry{Timestamp: now.Add(-1 * time.Second), Level: "info", Message: "recent"}
	futureEntry := LogEntry{Timestamp: now.Add(1 * time.Second), Level: "info", Message: "future"}

	buf.add(oldEntry)
	buf.add(recentEntry)
	buf.add(futureEntry)

	// Filter by 30 minutes ago - should get recent and future
	cutoff := now.Add(-30 * time.Minute)
	result := buf.getEntries(LogFilter{Since: cutoff})
	assert.Len(t, result, 2)

	// Verify order
	assert.Equal(t, "recent", result[0].Message)
	assert.Equal(t, "future", result[1].Message)
}

// TestLogBuffer_FilterByCount tests limiting result count.
func TestLogBuffer_FilterByCount(t *testing.T) {
	buf := newLogBuffer(10, true)

	// Add 10 entries
	for i := 0; i < 10; i++ {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   string(rune('a' + i)),
		})
	}

	// Request only 3
	result := buf.getEntries(LogFilter{Count: 3})
	assert.Len(t, result, 3)

	// Should get first 3 in chronological order
	assert.Equal(t, "a", result[0].Message)
	assert.Equal(t, "b", result[1].Message)
	assert.Equal(t, "c", result[2].Message)
}

// TestLogBuffer_ReverseOrder tests reverse chronological ordering.
func TestLogBuffer_ReverseOrder(t *testing.T) {
	buf := newLogBuffer(5, true)

	// Add entries
	for i := 0; i < 3; i++ {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   string(rune('a' + i)),
		})
	}

	// Get in reverse order
	result := buf.getEntries(LogFilter{Reverse: true})
	assert.Len(t, result, 3)

	// Should be in reverse order
	assert.Equal(t, "c", result[0].Message)
	assert.Equal(t, "b", result[1].Message)
	assert.Equal(t, "a", result[2].Message)
}

// TestLogBuffer_CombinedFilters tests multiple filters together.
func TestLogBuffer_CombinedFilters(t *testing.T) {
	buf := newLogBuffer(10, true)

	agent1 := uuid.New()
	agent2 := uuid.New()
	now := time.Now()

	// Add varied entries
	entries := []LogEntry{
		{Timestamp: now, Level: "info", Message: "i1", AgentID: agent1},
		{Timestamp: now, Level: "error", Message: "e1", AgentID: agent1},
		{Timestamp: now, Level: "info", Message: "i2", AgentID: agent2},
		{Timestamp: now, Level: "error", Message: "e2", AgentID: agent2},
		{Timestamp: now.Add(-1 * time.Hour), Level: "info", Message: "old", AgentID: agent1},
	}

	for _, entry := range entries {
		buf.add(entry)
	}

	// Filter: level=error, agent=agent1, count=1
	result := buf.getEntries(LogFilter{
		Level:   "error",
		AgentID: agent1,
		Count:   1,
	})
	assert.Len(t, result, 1)
	assert.Equal(t, "e1", result[0].Message)
	assert.Equal(t, agent1, result[0].AgentID)
}

// TestLogBuffer_ThreadSafety tests concurrent access to the buffer.
func TestLogBuffer_ThreadSafety(t *testing.T) {
	buf := newLogBuffer(1000, true)
	numGoroutines := 100
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Readers and writers

	// Writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				buf.add(LogEntry{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   string(rune(id)),
				})
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				_ = buf.getEntries(LogFilter{})
			}
		}()
	}

	wg.Wait()

	// Should have some entries (may be less than total due to overflow)
	assert.Greater(t, buf.count, 0)
	assert.LessOrEqual(t, buf.count, buf.maxSize)
}

// TestLogBuffer_OverflowLargeScale tests buffer overflow with many entries.
func TestLogBuffer_OverflowLargeScale(t *testing.T) {
	bufSize := 100
	buf := newLogBuffer(bufSize, true)

	// Add 3x buffer size
	for i := 0; i < bufSize*3; i++ {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   fmt.Sprintf("msg-%d", i),
		})
	}

	// Buffer should be full
	assert.Equal(t, bufSize, buf.count)

	// All entries should be the last 100 added
	result := buf.getEntries(LogFilter{})
	assert.Len(t, result, bufSize)

	// First entry should be from iteration 200 (300 - 100)
	// Verify we have the correct entries in buffer
	assert.Contains(t, result[0].Message, "200")
	assert.Contains(t, result[bufSize-1].Message, "299")
}

// TestLogBuffer_GetStats tests statistics retrieval.
func TestLogBuffer_GetStats(t *testing.T) {
	t.Run("Empty buffer", func(t *testing.T) {
		buf := newLogBuffer(100, true)
		stats := buf.getStats()

		assert.Equal(t, 100, stats["max_size"])
		assert.Equal(t, 0, stats["current"])
		assert.True(t, stats["enabled"].(bool))
		assert.False(t, stats["full"].(bool))
	})

	t.Run("Full buffer", func(t *testing.T) {
		buf := newLogBuffer(10, true)

		// Fill the buffer
		for i := 0; i < 10; i++ {
			buf.add(LogEntry{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "test",
			})
		}

		stats := buf.getStats()
		assert.Equal(t, 10, stats["max_size"])
		assert.Equal(t, 10, stats["current"])
		assert.True(t, stats["full"].(bool))
	})
}

// BenchmarkLogBuffer_Add benchmarks adding entries.
func BenchmarkLogBuffer_Add(b *testing.B) {
	buf := newLogBuffer(10000, true)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "benchmark test message",
		Fields:    map[string]interface{}{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.add(entry)
	}
}

// BenchmarkLogBuffer_GetEntries benchmarks retrieving entries.
func BenchmarkLogBuffer_GetEntries(b *testing.B) {
	buf := newLogBuffer(1000, true)

	// Fill buffer
	for i := 0; i < 1000; i++ {
		buf.add(LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "test message",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.getEntries(LogFilter{})
	}
}

// BenchmarkLogBuffer_ConcurrentAccess benchmarks concurrent reads and writes.
func BenchmarkLogBuffer_ConcurrentAccess(b *testing.B) {
	buf := newLogBuffer(1000, true)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between reads and writes
			buf.add(LogEntry{
				Timestamp: time.Now(),
				Level:     "info",
				Message:   "test",
			})
			_ = buf.getEntries(LogFilter{})
		}
	})
}
