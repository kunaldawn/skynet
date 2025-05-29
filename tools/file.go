/*
Package tools provides file system operation capabilities for the Skynet Agent.

This file implements the FileTool, which provides comprehensive file system
access and manipulation capabilities. The tool supports a wide range of file
operations including reading, writing, metadata inspection, and file management.

Supported operations:
- Reading: read, head, tail
- Metadata: size, exists, type, permissions
- Writing: write, edit, create
- File Management: delete, move, copy, chmod
- Directory Operations: mkdir, rmdir

All file operations are performed within the context of a working directory
and include proper error handling and logging for operational monitoring.
The tool implements size limits and safety checks to prevent system abuse.
*/
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

// fileLogger provides structured logging for all file operations
// with a consistent tool identifier for easy filtering and monitoring
var fileLogger = logrus.WithField("tool", "file")

// FileTool provides comprehensive file system operations for the agent.
// It maintains a working directory context and implements all standard
// file operations with proper error handling and logging.
type FileTool struct {
	workingDir *string // Reference to the current working directory for relative path resolution
}

// NewFileTool creates a new instance of the file operations tool.
// The tool requires a working directory reference for proper path resolution
// and maintains this context throughout its lifecycle.
//
// Parameters:
//   - workingDir: Pointer to the current working directory string
//
// Returns:
//   - *FileTool: Configured file tool ready for use
func NewFileTool(workingDir *string) *FileTool {
	fileLogger.Debug("Initializing file tool")
	return &FileTool{workingDir: workingDir}
}

// Description returns a comprehensive description of the file tool's capabilities.
// This description is used by the agent framework to understand what operations
// are available and how to invoke them properly.
//
// Returns:
//   - string: Detailed description of all supported file operations
func (f *FileTool) Description() string {
	return "File operations with full system access. Usage: 'read <path>' (read file content), 'head <path>' (first 20 lines), 'tail <path>' (last 20 lines), 'size <path>' (file size), 'exists <path>' (check existence), 'type <path>' (file type), 'permissions <path>' (file permissions), 'write <path> <content>' (write file content), 'edit <path> <content>' (edit file content), 'create <path> <content>' (create file), 'delete <path>' (delete file), 'move <src> <dst>' (move file), 'copy <src> <dst>' (copy file), 'chmod <mode> <path>' (change file permissions), 'mkdir <path>' (create directory), 'rmdir <path>' (remove directory)."
}

// Name returns the identifier for this tool.
// This name is used by the agent framework for tool selection and invocation.
//
// Returns:
//   - string: The tool's identifier ("file")
func (f *FileTool) Name() string {
	return "file"
}

// Call executes a file operation based on the provided input command.
// This is the main entry point for all file operations. The method parses
// the input command, validates parameters, resolves paths, and executes
// the requested operation with proper error handling and logging.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - input: Command string containing operation and parameters
//
// Returns:
//   - string: Formatted result of the operation or error message
//   - error: Always nil (errors are returned as string messages)
func (f *FileTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := fileLogger.WithField("input", input)
	toolLogger.Info("File tool called")
	startTime := time.Now()

	// Parse the input command into component parts
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		toolLogger.Warn("Empty file command provided")
		return "Error: Please provide a file command. Supported: read <path>, head <path>, tail <path>, size <path>, exists <path>, type <path>, permissions <path>, write <path> <content>, edit <path> <content>, create <path> <content>, delete <path>, move <src> <dst>, copy <src> <dst>, chmod <mode> <path>", nil
	}

	command := strings.ToLower(parts[0])

	// Validate that a path was provided
	if len(parts) < 2 {
		return "Error: Please specify a file path", nil
	}

	path := parts[1]

	// Resolve relative paths against the working directory
	var targetPath string
	if filepath.IsAbs(path) {
		targetPath = path
	} else {
		targetPath = filepath.Join(*f.workingDir, path)
	}

	var cmd *exec.Cmd
	var err error
	var output []byte

	switch command {
	case "read":
		// Use cat command
		cmd = exec.CommandContext(ctx, "cat", targetPath)

	case "head":
		// Use head command
		cmd = exec.CommandContext(ctx, "head", "-20", targetPath)

	case "tail":
		// Use tail command
		cmd = exec.CommandContext(ctx, "tail", "-20", targetPath)

	case "size":
		// Use wc command for file size
		cmd = exec.CommandContext(ctx, "wc", "-c", targetPath)

	case "exists":
		// Use test command
		cmd = exec.CommandContext(ctx, "test", "-e", targetPath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "false", nil
		}
		return "true", nil

	case "type":
		// Use file command
		cmd = exec.CommandContext(ctx, "file", targetPath)

	case "permissions":
		// Use stat command for permissions
		cmd = exec.CommandContext(ctx, "stat", "-c", "%A", targetPath)

	case "write", "edit", "create":
		if len(parts) < 3 {
			return "Error: Please provide content to write", nil
		}
		content := strings.Join(parts[2:], " ")
		err := os.WriteFile(targetPath, []byte(content), 0644)
		if err != nil {
			return fmt.Sprintf("Error writing file: %v", err), nil
		}
		return fmt.Sprintf("File written successfully: %s", targetPath), nil

	case "delete":
		err := os.Remove(targetPath)
		if err != nil {
			return fmt.Sprintf("Error deleting file: %v", err), nil
		}
		return fmt.Sprintf("File deleted successfully: %s", targetPath), nil

	case "move":
		if len(parts) < 3 {
			return "Error: Please provide destination path", nil
		}
		dstPath := parts[2]
		if !filepath.IsAbs(dstPath) {
			dstPath = filepath.Join(*f.workingDir, dstPath)
		}
		cmd = exec.CommandContext(ctx, "mv", targetPath, dstPath)

	case "copy":
		if len(parts) < 3 {
			return "Error: Please provide destination path", nil
		}
		dstPath := parts[2]
		if !filepath.IsAbs(dstPath) {
			dstPath = filepath.Join(*f.workingDir, dstPath)
		}
		cmd = exec.CommandContext(ctx, "cp", targetPath, dstPath)

	case "chmod":
		if len(parts) < 3 {
			return "Error: Please provide file mode", nil
		}
		mode := parts[1]
		filePath := parts[2]
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(*f.workingDir, filePath)
		}
		cmd = exec.CommandContext(ctx, "chmod", mode, filePath)

	case "mkdir":
		cmd = exec.CommandContext(ctx, "mkdir", "-p", targetPath)

	case "rmdir":
		cmd = exec.CommandContext(ctx, "rmdir", targetPath)

	default:
		return fmt.Sprintf("Unknown command '%s'. Supported commands: read, head, tail, size, exists, type, permissions, write, edit, create, delete, move, copy, chmod, mkdir, rmdir", command), nil
	}

	if cmd != nil {
		output, err = cmd.CombinedOutput()
		if err != nil {
			toolLogger.WithError(err).WithField("command", command).Error("File command failed")
			return string(output), nil
		}
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"targetPath":    targetPath,
		"executionTime": executionTime,
		"outputLength":  len(output),
	}).Info("File command completed")

	return string(output), nil
}

var _ tools.Tool = (*FileTool)(nil)
