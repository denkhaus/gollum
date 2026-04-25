// Package channel provides the channel abstraction layer for Gollum.
//
// LOG ROUTING BEHAVIOR:
//
// Channels receive log entries via the OnLog(entry) method. The routing
// behavior depends on the channel type and use case:
//
// SINGLE-SESSION CHANNELS (e.g., TUI):
// - Receive all logs for display in the UI
// - Logs are not session-scoped
// - Example: TUI shows all system logs in a dedicated panel
//
// MULTI-SESSION CHANNELS (e.g., ACP):
//   - MUST route logs to specific sessions when entry.SessionID is set
//   - MAY broadcast system-wide logs (no session ID) to all active sessions
//   - Example: ACP forwards session-specific logs to that session only,
//     but broadcasts agent lifecycle events to all sessions
//
// LOG ENTRY ROUTING:
// - entry.ChannelID determines which channel receives the log
// - entry.SessionID provides session-specific routing (optional)
// - Channels MUST handle logs even if no session is active
//
// IMPLEMENTATION NOTES:
// - Use non-blocking sends to prevent log system deadlocks
// - Filter/handle logs appropriately for the channel type
// - Document channel-specific log behavior in channel documentation
package channel
