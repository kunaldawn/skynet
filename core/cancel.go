/*
Package core provides execution cancellation management for the Skynet Agent application.

This file implements the CancelManager, which tracks and manages running agent
executions to provide cancellation capabilities. This is essential for allowing
users to stop long-running or infinite loop operations gracefully.

The cancellation system provides:
- Thread-safe tracking of active executions
- Context-based cancellation for clean shutdown
- Execution lifecycle management
- Active execution monitoring and reporting

The system integrates with Go's context cancellation patterns to ensure
proper resource cleanup and responsive user control over agent operations.
*/
package core

import (
	"context"
	"sync"
)

// CancelManager tracks running agent executions and provides cancellation capabilities.
// It maintains a thread-safe registry of active executions with their associated
// cancellation functions, enabling users to stop operations that may be running
// indefinitely or taking too long to complete.
//
// The manager integrates with Go's context cancellation patterns to ensure
// clean shutdown and proper resource cleanup when executions are cancelled.
type CancelManager struct {
	executions map[string]context.CancelFunc // Map of execution ID to cancellation function
	mutex      sync.RWMutex                  // Read-write mutex for thread-safe access to the executions map
}

// NewCancelManager creates and initializes a new cancel manager instance.
// The manager starts with an empty execution registry and is ready to
// track new executions immediately.
//
// Returns:
//   - *CancelManager: Initialized cancel manager ready for use
func NewCancelManager() *CancelManager {
	return &CancelManager{
		executions: make(map[string]context.CancelFunc),
	}
}

// AddExecution registers a new execution with its cancellation function.
// This method should be called when starting a new agent execution to
// enable cancellation capabilities. The execution ID should be unique
// and the cancel function should properly clean up all associated resources.
//
// Parameters:
//   - executionID: Unique identifier for the execution
//   - cancel: Context cancellation function that will stop the execution
func (cm *CancelManager) AddExecution(executionID string, cancel context.CancelFunc) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.executions[executionID] = cancel
}

// RemoveExecution removes a completed or cancelled execution from tracking.
// This method should be called when an execution completes naturally or
// after it has been successfully cancelled to prevent memory leaks and
// maintain accurate tracking of active executions.
//
// Parameters:
//   - executionID: Unique identifier of the execution to remove
func (cm *CancelManager) RemoveExecution(executionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.executions, executionID)
}

// CancelExecution attempts to cancel a running execution by ID.
// This method looks up the execution's cancellation function and invokes it
// if the execution is found. The execution is automatically removed from
// tracking after cancellation.
//
// Parameters:
//   - executionID: Unique identifier of the execution to cancel
//
// Returns:
//   - bool: true if the execution was found and cancelled, false if not found
func (cm *CancelManager) CancelExecution(executionID string) bool {
	// Use read lock to check existence and get cancel function
	cm.mutex.RLock()
	cancel, exists := cm.executions[executionID]
	cm.mutex.RUnlock()

	if exists {
		// Cancel the execution using its context cancellation function
		cancel()
		// Remove from tracking after successful cancellation
		cm.RemoveExecution(executionID)
		return true
	}
	return false
}

// GetActiveExecutions returns a list of all currently active execution IDs.
// This method provides visibility into what executions are currently running
// and can be used for monitoring, debugging, or administrative purposes.
//
// Returns:
//   - []string: Slice containing all active execution IDs
func (cm *CancelManager) GetActiveExecutions() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Create a slice with appropriate capacity to avoid reallocations
	executions := make([]string, 0, len(cm.executions))
	for id := range cm.executions {
		executions = append(executions, id)
	}
	return executions
}
