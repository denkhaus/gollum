package tui

// This file handles agent execution events for the TUI.

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// handleAgentCompleteMsg handles the completion of agent execution.
func (m Model) handleAgentCompleteMsg(msg agentCompleteMsg) (Model, tea.Cmd) {
	m.agentExecuting = false
	m.agentStartTime = time.Time{} // Clear start time
	m.textInput.Focus()

	if msg.err != nil {
		m = m.handleAgentError(msg.err)
	} else if msg.response != nil {
		m = m.handleAgentResponse(msg.response)
	}

	// Restore preserved input if user canceled during execution
	m = m.restorePreservedInput()

	// Return Blink command to restore cursor blinking
	return m, textinput.Blink
}

// handleNewMessageMsg handles new messages from AgentMessenger.
func (m Model) handleNewMessageMsg(msg newMessageMsg) (Model, tea.Cmd) {
	m.addMessage(msg.message)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom()
	return m, m.waitForMessages()
}

// handleAgentError handles agent execution errors.
func (m Model) handleAgentError(err error) Model {
	m.err = err

	// Add error message to messages
	errorMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeError,
		Content:   err.Error(),
		Timestamp: time.Now(),
	}
	m.addMessage(errorMsg)

	// Update viewport with error message
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom()

	return m
}

// handleAgentResponse handles successful agent responses.
func (m Model) handleAgentResponse(response *gollem.ExecuteResponse) Model {
	// Only add response texts if NOT using AgentMessenger
	// When AgentMessenger is active (messageChan configured), messages are sent
	// via the channel during execution, avoiding duplicates
	if m.shouldAddResponseTexts() {
		// Add response texts as individual messages (legacy mode)
		for _, text := range response.Texts {
			agentMsg := channel.Message{
				ID:        uuid.New(),
				Type:      channel.MessageTypeAgentChat,
				Content:   text,
				Timestamp: time.Now(),
				AgentID:   uuid.Nil, // Will be set by agent messenger
				AgentRole: "",
			}
			m.addMessage(agentMsg)
		}
		// Update viewport with new messages
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoBottom()
	}
	// When messageChan is active, messages were already added via newMessageMsg
	// during execution, so no viewport update needed here
	return m
}

// restorePreservedInput restores the preserved input after cancellation.
func (m Model) restorePreservedInput() Model {
	if m.cancelRequested && m.preservedInput != "" {
		m.textInput.SetValue(m.preservedInput)
		m.textInput.CursorEnd()
		m.preservedInput = ""
		m.cancelRequested = false
	}
	return m
}

// shouldAddResponseTexts determines if response texts should be added.
func (m Model) shouldAddResponseTexts() bool {
	return m.messageChan == nil
}

// prepareAgentExecution prepares the model for agent execution.
func (m Model) prepareAgentExecution() Model {
	// Set agent execution state BEFORE creating the command
	// (this is needed because commands can't modify the model)
	m.agentExecuting = true
	m.agentStartTime = time.Now()
	m.textInput.Blur()
	return m
}

// createAgentCommand creates a command to execute the agent.
func (m Model) createAgentCommand(input string) tea.Cmd {
	// Create a per-request context that can be canceled
	// Defensive: ensure we always have a cancellable context
	var reqCtx context.Context
	var cancel context.CancelFunc

	if m.ctx != nil {
		reqCtx, cancel = context.WithCancel(m.ctx)
	} else {
		reqCtx, cancel = context.WithCancel(context.Background())
	}
	m.currentCancel = cancel

	// Return the command that will execute the agent
	return func() tea.Msg {
		defer cancel()
		m.currentCancel = nil

		response, err := m.agent.Execute(reqCtx, input)
		return agentCompleteMsg{response: response, err: err}
	}
}
