// Package logger provides in-memory circular buffer for log entries.
package logger

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LogEntry represents a single log entry in the session buffer.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	AgentID   uuid.UUID              `json:"agent_id,omitempty"`
	Sequence  int64                  `json:"sequence"` // Monotonically increasing sequence number
}

// LogFilter defines filtering options for retrieving log entries.
type LogFilter struct {
	Level    string    // Filter by log level (debug, info, warn, error)
	AgentID  uuid.UUID // Filter by specific agent ID
	SinceSeq *int64    // Filter entries with sequence > this (exclusive), nil = no filter
	Count    int       // Maximum number of entries to return (0 = all)
	Reverse  bool      // If true, return entries in reverse chronological order
}

// logBuffer implements a thread-safe circular buffer for log entries.
type logBuffer struct {
	entries   []LogEntry
	maxSize   int
	mutex     sync.RWMutex
	headIndex int
	count     int
	enabled   bool
	nextSeq   int64 // Next sequence number to assign
}

// newLogBuffer creates a new circular buffer for log entries.
func newLogBuffer(maxSize int, enabled bool) *logBuffer {
	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}

	return &logBuffer{
		entries:   make([]LogEntry, maxSize),
		maxSize:   maxSize,
		headIndex: 0,
		count:     0,
		enabled:   enabled,
	}
}

// add adds a new log entry to the buffer.
// If the buffer is full, the oldest entry is overwritten.
func (b *logBuffer) add(entry LogEntry) {
	if !b.enabled {
		return
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Assign sequence number atomically
	entry.Sequence = b.nextSeq
	b.nextSeq++

	// Store the entry at the current head position
	b.entries[b.headIndex] = entry

	// Move head to the next position (wrapping around)
	b.headIndex = (b.headIndex + 1) % b.maxSize

	// If we haven't filled the buffer yet, increment count
	if b.count < b.maxSize {
		b.count++
	}
}

// getEntries retrieves log entries matching the given filter.
func (b *logBuffer) getEntries(filter LogFilter) []LogEntry {
	if !b.enabled {
		return []LogEntry{}
	}

	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Collect all entries (in chronological order)
	allEntries := make([]LogEntry, 0, b.count)
	for i := 0; i < b.count; i++ {
		// Calculate the actual index based on head position and count
		// This handles the circular nature of the buffer
		idx := (b.headIndex - b.count + i + b.maxSize) % b.maxSize
		allEntries = append(allEntries, b.entries[idx])
	}

	// Apply filters
	filtered := b.applyFilters(allEntries, filter)

	// Apply reverse if requested
	if filter.Reverse {
		// Reverse the slice
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	return filtered
}

// applyFilters applies the filter criteria to a list of entries.
func (b *logBuffer) applyFilters(entries []LogEntry, filter LogFilter) []LogEntry {
	result := entries

	// Filter by sequence number (preferred over timestamp for deduplication)
	// SinceSeq filters entries with sequence > SinceSeq (exclusive)
	// nil means no filtering
	if filter.SinceSeq != nil {
		filtered := make([]LogEntry, 0)
		for _, entry := range result {
			if entry.Sequence > *filter.SinceSeq {
				filtered = append(filtered, entry)
			}
		}
		result = filtered
	}

	// Filter by level
	if filter.Level != "" {
		filtered := make([]LogEntry, 0)
		for _, entry := range result {
			if entry.Level == filter.Level {
				filtered = append(filtered, entry)
			}
		}
		result = filtered
	}

	// Filter by agent ID
	if filter.AgentID != uuid.Nil {
		filtered := make([]LogEntry, 0)
		for _, entry := range result {
			if entry.AgentID == filter.AgentID {
				filtered = append(filtered, entry)
			}
		}
		result = filtered
	}

	// Apply count limit
	if filter.Count > 0 && len(result) > filter.Count {
		result = result[:filter.Count]
	}

	return result
}

// zapFieldsToMap converts zap.Field slice to a map.
func zapFieldsToMap(fields []zap.Field) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range fields {
		result[field.Key] = field.Interface
	}
	return result
}

// getStats returns statistics about the buffer.
func (b *logBuffer) getStats() map[string]interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return map[string]interface{}{
		"max_size": b.maxSize,
		"current":  b.count,
		"enabled":  b.enabled,
		"full":     b.count == b.maxSize,
	}
}
