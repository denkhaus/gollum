package mapping

import (
	"encoding/json"
	"os"
	"sync"
)

// Manager handles session-to-branch mapping
type Manager struct {
	mu          sync.RWMutex
	mappingFile string
}

// SessionMapping represents the mapping structure
type SessionMapping struct {
	Sessions map[string]string `json:"sessions"` // session_id → issue_name
	Issues   map[string]*IssueInfo `json:"issues"`   // issue_name → IssueInfo
}

// IssueInfo tracks issue state
type IssueInfo struct {
	Status        string   `json:"status"`
	ActiveSession string   `json:"active_session"`
	Branch        string   `json:"branch"`
	History       []string `json:"history"`
}

// NewManager creates a new mapping manager
func NewManager(mappingFile string) *Manager {
	return &Manager{
		mappingFile: mappingFile,
	}
}

// GetBranchForSession returns the branch name for a given session
func (m *Manager) GetBranchForSession(sessionID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping, err := m.load()
	if err != nil {
		return "", err
	}

	if branch, exists := mapping.Sessions[sessionID]; exists {
		return branch, nil
	}

	return "", os.ErrNotExist
}

// RegisterSession registers a session for an issue
func (m *Manager) RegisterSession(sessionID, issueName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapping, err := m.load()
	if err != nil {
		mapping = &SessionMapping{
			Sessions: make(map[string]string),
			Issues:   make(map[string]*IssueInfo),
		}
	}

	// Update session mapping
	mapping.Sessions[sessionID] = issueName

	// Update or create issue info
	if issueInfo, exists := mapping.Issues[issueName]; exists {
		issueInfo.ActiveSession = sessionID
		issueInfo.History = append(issueInfo.History, sessionID)
	} else {
		mapping.Issues[issueName] = &IssueInfo{
			Status:        "in-progress",
			ActiveSession: sessionID,
			Branch:        issueName,
			History:       []string{sessionID},
		}
	}

	return m.save(mapping)
}

// GetMapping returns the current mapping
func (m *Manager) GetMapping() (*SessionMapping, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.load()
}

func (m *Manager) load() (*SessionMapping, error) {
	mapping := &SessionMapping{
		Sessions: make(map[string]string),
		Issues:   make(map[string]*IssueInfo),
	}

	data, err := os.ReadFile(m.mappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return mapping, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

func (m *Manager) save(mapping *SessionMapping) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.mappingFile, data, 0644)
}
