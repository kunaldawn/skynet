/*
Package tools provides Docker container and image management capabilities for the Skynet Agent.

This file implements the DockerTool, which provides comprehensive Docker functionality
including container lifecycle management, image operations, monitoring, and debugging.
The tool acts as a bridge between the agent and the Docker daemon, enabling full
Docker command-line functionality through the agent interface.

Supported operations:
- Container Management: ps, logs, inspect, stats, run, stop, start, rm
- Image Management: images, build, pull, push, rmi
- System Operations: version, info, system commands
- All standard Docker CLI commands with proper formatting and error handling

The tool provides enhanced formatting for common read-only operations while
supporting the full Docker command set for advanced operations.
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

// dockerLogger provides structured logging for all Docker operations
// with a consistent tool identifier for easy filtering and monitoring
var dockerLogger = logrus.WithField("tool", "docker")

// DockerTool provides comprehensive Docker container and image management capabilities.
// It wraps the Docker CLI to provide agent-accessible container operations with
// enhanced formatting, error handling, and logging for operational monitoring.
type DockerTool struct{}

// NewDockerTool creates a new instance of the Docker management tool.
// The tool requires Docker to be installed and accessible in the system PATH.
//
// Returns:
//   - *DockerTool: Configured Docker tool ready for use
func NewDockerTool() *DockerTool {
	dockerLogger.Debug("Initializing docker tool")
	return &DockerTool{}
}

// Description returns a comprehensive description of the Docker tool's capabilities.
// This description is used by the agent framework to understand what Docker
// operations are available and how to invoke them properly.
//
// Returns:
//   - string: Detailed description of all supported Docker operations
func (d *DockerTool) Description() string {
	return "Manage Docker containers and images. Supports all Docker commands including: 'ps' (list containers), 'images' (list images), 'logs <container>' (view logs), 'inspect <container>' (inspect container), 'stats' (container stats), 'version' (docker version), 'run', 'stop', 'start', 'rm', 'rmi', 'build', 'pull', 'push', etc. Full Docker functionality is available."
}

// Name returns the identifier for this tool.
// This name is used by the agent framework for tool selection and invocation.
//
// Returns:
//   - string: The tool's identifier ("docker")
func (d *DockerTool) Name() string {
	return "docker"
}

// Call executes a Docker command based on the provided input.
// This is the main entry point for all Docker operations. The method parses
// the input command, validates Docker availability, and executes the requested
// operation with proper error handling, timeout management, and result formatting.
//
// The method provides enhanced formatting for common read-only operations like
// ps, images, logs, inspect, stats, and version, while supporting the full
// Docker command set for advanced operations.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - input: Docker command string (e.g., "ps -a", "logs container_name")
//
// Returns:
//   - string: Formatted result of the Docker operation or error message
//   - error: Always nil (errors are returned as string messages)
func (d *DockerTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := dockerLogger.WithField("input", input)
	toolLogger.Info("Docker tool called")
	startTime := time.Now()

	// Parse the input command into component parts
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		toolLogger.Warn("Empty docker command provided")
		return "Error: Please provide a docker command. All Docker commands are supported.", nil
	}

	command := strings.ToLower(parts[0])

	toolLogger.WithField("command", command).Debug("Docker command validated")

	// Verify Docker availability before attempting operations
	if err := exec.Command("docker", "--version").Run(); err != nil {
		toolLogger.WithError(err).Error("Docker not available")
		return "Error: Docker is not installed or not accessible", nil
	}

	// Execute command with timeout to prevent hanging operations
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "docker", parts...)

	// Execute the Docker command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithFields(logrus.Fields{
			"command": command,
			"output":  string(output),
		}).Error("Docker command failed")

		// Handle timeout specifically
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "Error: Docker command timed out after 30 seconds", nil
		}

		return string(output), nil
	}

	// Log execution metrics for monitoring
	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Docker command completed")

	return string(output), nil
}

// Ensure DockerTool implements the tools.Tool interface
var _ tools.Tool = (*DockerTool)(nil)
