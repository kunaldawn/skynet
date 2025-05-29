/*
Package core provides session memory management for the Skynet Agent application.

This file implements a thread-safe, in-memory storage system for managing
conversation sessions. It provides conversation continuity across multiple
interactions while implementing automatic cleanup to prevent memory leaks.

Key components:
- ChatMessage: Individual conversation messages with metadata
- ChatSession: Complete conversation context with thread-safe operations
- MemoryStore: Centralized session management with automatic cleanup

The memory system is designed for high-concurrency scenarios with proper
locking mechanisms and automatic resource management.
*/
package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ChatMessage represents a single message in a conversation between user and assistant.
// Each message includes role identification, content, and timing information for
// proper conversation context reconstruction.
type ChatMessage struct {
	Role      string    `json:"role"`      // Message sender: "user" or "assistant"
	Content   string    `json:"content"`   // The actual message text content
	Timestamp time.Time `json:"timestamp"` // When the message was created (for debugging and analytics)
}

// ChatSession represents a complete conversation session with memory persistence.
// Sessions maintain conversation history and provide thread-safe access to message
// operations. Each session has a unique identifier and tracks its lifecycle.
type ChatSession struct {
	ID       string        `json:"id"`       // Unique session identifier for client reference
	Messages []ChatMessage `json:"messages"` // Ordered list of conversation messages
	Created  time.Time     `json:"created"`  // Session creation timestamp
	Updated  time.Time     `json:"updated"`  // Last activity timestamp for cleanup decisions
	mutex    sync.RWMutex  // Read-write mutex for thread-safe concurrent access
}

// MemoryStore manages multiple chat sessions with automatic lifecycle management.
// It provides centralized storage, retrieval, and cleanup of conversation sessions
// while ensuring thread safety and preventing memory leaks through automatic expiration.
type MemoryStore struct {
	sessions        map[string]*ChatSession // Map of session ID to session objects
	mutex           sync.RWMutex            // Read-write mutex for thread-safe map operations
	maxAge          time.Duration           // Maximum age for sessions before cleanup eligibility
	cleanupInterval time.Duration           // How frequently to run automatic cleanup
	logger          *logrus.Logger          // Structured logger for operational monitoring
}

// NewMemoryStore creates and initializes a new memory store with automatic cleanup.
// The store begins monitoring and cleaning up expired sessions immediately upon creation.
//
// Parameters:
//   - maxAge: Duration after which inactive sessions become eligible for cleanup
//   - cleanupInterval: How often to run the cleanup process
//   - logger: Logger instance for operational monitoring and debugging
//
// Returns:
//   - *MemoryStore: Configured memory store ready for use
func NewMemoryStore(maxAge time.Duration, cleanupInterval time.Duration, logger *logrus.Logger) *MemoryStore {
	store := &MemoryStore{
		sessions:        make(map[string]*ChatSession),
		maxAge:          maxAge,
		cleanupInterval: cleanupInterval,
		logger:          logger,
	}

	// Start background cleanup goroutine for automatic session management
	go store.cleanupExpiredSessions()

	return store
}

// generateSessionID creates a cryptographically secure unique session identifier.
// Uses crypto/rand for security when available, falls back to timestamp-based ID
// if random generation fails to ensure reliable operation.
//
// Returns:
//   - string: Unique session identifier with "session_" prefix
func generateSessionID() string {
	bytes := make([]byte, 16) // 16 bytes = 128 bits of entropy
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return "session_" + hex.EncodeToString(bytes)
}

// GetOrCreateSession retrieves an existing session or creates a new one if needed.
// This is the primary method for session management, ensuring clients always
// receive a valid session regardless of whether one previously existed.
//
// Parameters:
//   - sessionID: Existing session ID, or empty string to create new session
//
// Returns:
//   - *ChatSession: Valid session object (existing or newly created)
func (m *MemoryStore) GetOrCreateSession(sessionID string) *ChatSession {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate new session ID if none provided
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	session, exists := m.sessions[sessionID]
	if !exists {
		// Create new session with empty message history
		session = &ChatSession{
			ID:       sessionID,
			Messages: make([]ChatMessage, 0),
			Created:  time.Now(),
			Updated:  time.Now(),
		}
		m.sessions[sessionID] = session
		m.logger.WithField("sessionID", sessionID).Info("Created new chat session")
	} else {
		// Update access time for existing session
		session.Updated = time.Now()
	}

	return session
}

// GetSession retrieves an existing session without creating a new one.
// This method is useful for checking session existence or retrieving
// sessions for read-only operations.
//
// Parameters:
//   - sessionID: The session identifier to retrieve
//
// Returns:
//   - *ChatSession: The session object if found
//   - bool: Whether the session exists
func (m *MemoryStore) GetSession(sessionID string) (*ChatSession, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[sessionID]
	if exists {
		// Update access time when session is retrieved
		session.Updated = time.Now()
	}
	return session, exists
}

// DeleteSession removes a session from the store by ID.
// This method provides explicit session cleanup for administrative
// operations or user-requested session termination.
//
// Parameters:
//   - sessionID: The session identifier to delete
//
// Returns:
//   - bool: Whether the session existed and was deleted
func (m *MemoryStore) DeleteSession(sessionID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.sessions[sessionID]
	if exists {
		delete(m.sessions, sessionID)
		m.logger.WithField("sessionID", sessionID).Info("Session deleted")
	}
	return exists
}

// GetAllSessions returns a snapshot of all current sessions.
// This method is primarily used for administrative monitoring and
// debugging purposes. The returned slice is a copy to prevent external modification.
//
// Returns:
//   - []*ChatSession: Slice containing all current sessions
func (m *MemoryStore) GetAllSessions() []*ChatSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessions := make([]*ChatSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// AddMessage appends a new message to the session's conversation history.
// This method ensures thread-safe message addition and updates the session's
// last activity timestamp for cleanup management.
//
// Parameters:
//   - role: The message sender ("user" or "assistant")
//   - content: The message text content
func (s *ChatSession) AddMessage(role, content string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	message := ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	s.Messages = append(s.Messages, message)
	s.Updated = time.Now()
}

// GetRecentMessages returns the most recent messages up to a specified limit.
// This method is essential for maintaining conversation context without
// overwhelming the AI model with excessive history.
//
// Parameters:
//   - limit: Maximum number of recent messages to return
//
// Returns:
//   - []ChatMessage: Slice of recent messages in chronological order
func (s *ChatSession) GetRecentMessages(limit int) []ChatMessage {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(s.Messages) <= limit {
		return s.Messages
	}

	// Return the last 'limit' messages
	return s.Messages[len(s.Messages)-limit:]
}

// ClearMessages removes all messages from the session.
// This method provides a way to reset conversation context while
// maintaining the session identity. Returns the count of cleared messages for logging.
//
// Returns:
//   - int: Number of messages that were cleared
func (s *ChatSession) ClearMessages() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	messageCount := len(s.Messages)
	s.Messages = make([]ChatMessage, 0)
	s.Updated = time.Now()
	return messageCount
}

// GetConversationContext formats recent messages for inclusion in AI prompts.
// This method creates a human-readable conversation context that can be
// included in prompts to provide the AI with conversation history.
//
// Parameters:
//   - limit: Maximum number of recent messages to include
//
// Returns:
//   - string: Formatted conversation context ready for prompt inclusion
func (s *ChatSession) GetConversationContext(limit int) string {
	messages := s.GetRecentMessages(limit)
	if len(messages) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("Previous conversation context:\n")

	// Format each message with appropriate role labels
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			context.WriteString(fmt.Sprintf("Human: %s\n", msg.Content))
		case "assistant":
			context.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		}
	}

	context.WriteString("\nCurrent conversation:\n")
	return context.String()
}

// cleanupExpiredSessions runs as a background goroutine to automatically remove old sessions.
// This prevents memory leaks by periodically removing sessions that have been inactive
// for longer than the configured maximum age. The cleanup process is logged for monitoring.
func (m *MemoryStore) cleanupExpiredSessions() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		now := time.Now()
		expired := make([]string, 0)

		// Identify sessions that have exceeded the maximum age
		for id, session := range m.sessions {
			if now.Sub(session.Updated) > m.maxAge {
				expired = append(expired, id)
			}
		}

		// Remove expired sessions from the store
		for _, id := range expired {
			delete(m.sessions, id)
		}

		// Log cleanup results for operational monitoring
		if len(expired) > 0 {
			m.logger.WithFields(logrus.Fields{
				"expiredSessions":   len(expired),
				"remainingSessions": len(m.sessions),
				"cleanupInterval":   m.cleanupInterval,
			}).Info("Cleaned up expired chat sessions")
		}

		m.mutex.Unlock()
	}
}

// GetSessionStats returns operational statistics about stored sessions.
// This method provides insights into memory usage and conversation volume
// for monitoring and capacity planning purposes.
//
// Returns:
//   - map[string]interface{}: Statistics including session and message counts
func (m *MemoryStore) GetSessionStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	totalMessages := 0
	// Count total messages across all sessions
	for _, session := range m.sessions {
		session.mutex.RLock()
		totalMessages += len(session.Messages)
		session.mutex.RUnlock()
	}

	return map[string]interface{}{
		"totalSessions": len(m.sessions),
		"totalMessages": totalMessages,
	}
}
