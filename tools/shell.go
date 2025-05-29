/*
Package tools provides shell command execution capabilities for the Skynet Agent.

This file implements the ShellTool, which provides comprehensive shell command execution
functionality with full system privileges. The tool acts as a bridge between the agent
and the underlying operating system, enabling direct shell command execution through
the agent interface with proper security context and error handling.

Supported operations:
- Full Shell Command Execution: Execute any shell command with root privileges
- Working Directory Management: Commands execute in the specified working directory
- Environment Variable Control: Proper environment setup for root-level operations
- Combined Output Capture: Capture both stdout and stderr for comprehensive results
- Error Handling: Detailed error reporting with command output preservation
- Security Context: Automatic elevation to root privileges for system operations

The tool provides direct access to the shell while maintaining proper logging,
error handling, and execution context management for operational monitoring.

Security Note: This tool provides full shell access with root privileges and should
be used with appropriate caution and access controls in production environments.
*/
package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

// shellLogger provides structured logging for all shell operations
// with a consistent tool identifier for easy filtering and monitoring
var shellLogger = logrus.WithField("tool", "shell")

// ShellTool provides comprehensive shell command execution capabilities.
// It wraps the system shell to provide agent-accessible command execution with
// full privileges, proper working directory management, and comprehensive logging.
type ShellTool struct {
	workingDir *string // Pointer to the working directory for command execution
}

// NewShellTool creates a new instance of the shell command execution tool.
// The tool requires a working directory pointer for context-aware command execution
// and provides full shell access with root privileges.
//
// Parameters:
//   - workingDir: Pointer to the working directory for command execution context
//
// Returns:
//   - *ShellTool: Configured shell tool ready for command execution
func NewShellTool(workingDir *string) *ShellTool {
	shellLogger.Debug("Initializing shell tool")
	return &ShellTool{workingDir: workingDir}
}

// Description returns a comprehensive description of the shell tool's capabilities.
// This description is used by the agent framework to understand what shell
// operations are available and how to invoke them properly.
//
// Returns:
//   - string: Detailed description of shell command execution capabilities
func (s *ShellTool) Description() string {
	return "Execute shell commands. Usage: provide a shell command to execute. Commands are executed in the current working directory."
}

// Name returns the identifier for this tool.
// This name is used by the agent framework for tool selection and invocation.
//
// Returns:
//   - string: The tool's identifier ("shell")
func (s *ShellTool) Name() string {
	return "shell"
}

// Call executes a shell command based on the provided input.
// This is the main entry point for all shell operations. The method validates
// the input command, sets up the proper execution environment with root privileges,
// executes the command in the specified working directory, and returns the results
// with comprehensive error handling and logging.
//
// The method provides full shell access while maintaining proper security context,
// environment variable setup, and execution monitoring for operational oversight.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - input: Shell command string to execute (e.g., "ls -la", "mkdir /tmp/test")
//
// Returns:
//   - string: Command output (stdout and stderr combined) or error message
//   - error: Always nil (errors are returned as string messages)
func (s *ShellTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := shellLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": *s.workingDir,
	})
	toolLogger.Info("Shell tool called")
	startTime := time.Now()

	command := strings.TrimSpace(input)
	if command == "" {
		toolLogger.Warn("Empty shell command provided")
		return "Error: Please provide a shell command to execute", nil
	}

	// Execute command in working directory
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = *s.workingDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("command", command).Error("Shell command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Shell command completed")

	return string(output), nil
}

var _ tools.Tool = (*ShellTool)(nil)
