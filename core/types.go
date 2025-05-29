/*
Package core contains the fundamental data types and structures used throughout
the Skynet Agent application.

This file defines the core request/response types for the chat API, streaming
communication, and execution control. These types serve as the contract between
the client and server, ensuring consistent data exchange formats.

Key type categories:
- Chat API types (ChatRequest, ChatResponse)
- Real-time streaming types (StreamMessage)
- Execution control types (StopRequest, StopResponse)
*/
package core

// ChatRequest represents incoming chat requests from clients.
// This is the primary input structure for chat interactions with the agent.
type ChatRequest struct {
	Message   string `json:"message"`             // The user's message/query to the agent
	SessionID string `json:"sessionId,omitempty"` // Optional session ID for conversation memory continuity
	Debug     bool   `json:"debug,omitempty"`     // Enable debug mode for internal chain streaming and detailed logs
}

// ChatResponse represents the final response returned by the chat API.
// This contains the agent's response along with session management information.
type ChatResponse struct {
	Response  string `json:"response"`  // The agent's final response message
	SessionID string `json:"sessionId"` // Session ID returned to client for maintaining conversation context
}

// StreamMessage represents real-time streaming messages sent to clients via WebSocket.
// This enables live updates during agent execution, including tool usage, thinking processes,
// and intermediate results. The Type field determines how the client should handle each message.
type StreamMessage struct {
	Type      string                 `json:"type"`                // Message type: "thinking", "tool", "response", "error", "debug", "chain_start", "chain_step", "llm_call", "agent_action", "session", "execution_started", "stopped"
	Content   string                 `json:"content"`             // Main message content or description
	Tool      string                 `json:"tool,omitempty"`      // Name of the tool being executed (when Type is "tool")
	Complete  bool                   `json:"complete"`            // Whether this message represents completion of an operation
	Debug     bool                   `json:"debug,omitempty"`     // Whether this is a debug message (only sent when debug mode is enabled)
	Iteration int                    `json:"iteration,omitempty"` // Current iteration number in multi-step processes
	Step      string                 `json:"step,omitempty"`      // Current step identifier in the agent execution chain
	Details   map[string]interface{} `json:"details,omitempty"`   // Additional structured data for debugging and detailed logging
}

// StopRequest represents a client request to stop an ongoing agent execution.
// This allows users to cancel long-running operations or infinite loops.
type StopRequest struct {
	ExecutionID string `json:"executionId"` // Unique identifier of the execution to stop
}

// StopResponse represents the server's response to a stop request.
// This confirms whether the stop operation was successful and provides status information.
type StopResponse struct {
	Success bool   `json:"success"` // Whether the stop request was processed successfully
	Message string `json:"message"` // Human-readable message describing the result
	Stopped bool   `json:"stopped"` // Whether the execution was actually stopped (may already be completed)
}
