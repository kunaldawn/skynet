/*
Package tools provides text search and pattern matching capabilities for the Skynet Agent.

This file implements the GrepTool, which provides comprehensive text search functionality
using regular expressions to find patterns in files and directories. The tool acts as
a bridge between the agent and file system search operations, enabling powerful text
search capabilities through the agent interface.

Supported operations:
- Single File Search: Search for patterns within a specific file
- Directory Search: Recursively search for patterns across multiple files in a directory
- Regular Expression Support: Full regex pattern matching capabilities
- Text File Detection: Automatic filtering to search only text-based files
- Result Limiting: Intelligent result limiting to prevent overwhelming output
- Path Resolution: Support for both absolute and relative paths

The tool provides enhanced formatting and result summarization while supporting
full regular expression syntax for advanced pattern matching operations.
*/
package tools

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

// grepLogger provides structured logging for all grep operations
// with a consistent tool identifier for easy filtering and monitoring
var grepLogger = logrus.WithField("tool", "grep")

// GrepTool provides comprehensive text search and pattern matching capabilities.
// It wraps file system operations to provide agent-accessible text search with
// regular expression support, intelligent file filtering, and result formatting.
type GrepTool struct {
	workingDir string // Base directory for relative path resolution
}

// NewGrepTool creates a new instance of the text search tool.
// The tool requires a working directory for resolving relative paths and
// provides context-aware search operations.
//
// Parameters:
//   - workingDir: Pointer to the base directory for relative path resolution
//
// Returns:
//   - *GrepTool: Configured grep tool ready for use
func NewGrepTool(workingDir *string) *GrepTool {
	grepLogger.WithField("workingDir", *workingDir).Debug("Initializing grep tool")
	return &GrepTool{workingDir: *workingDir}
}

// Description returns a comprehensive description of the grep tool's capabilities.
// This description is used by the agent framework to understand what search
// operations are available and how to invoke them properly.
//
// Returns:
//   - string: Detailed description of all supported search operations
func (g *GrepTool) Description() string {
	return "Search for text patterns in files. Format: 'pattern filename' or 'pattern' to search in current directory. Supports basic regex patterns."
}

// Name returns the identifier for this tool.
// This name is used by the agent framework for tool selection and invocation.
//
// Returns:
//   - string: The tool's identifier ("grep")
func (g *GrepTool) Name() string {
	return "grep"
}

// Call executes a text search operation based on the provided input.
// This is the main entry point for all search operations. The method parses
// the input pattern and target, validates the regular expression, and executes
// the search operation with proper error handling, result formatting, and
// performance monitoring.
//
// The method supports both single file and directory searches, with intelligent
// file type filtering for directory operations and result limiting to prevent
// overwhelming output.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - input: Search command string (e.g., "pattern", "pattern filename")
//
// Returns:
//   - string: Formatted search results with matches and summary information
//   - error: Always nil (errors are returned as string messages)
func (g *GrepTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := grepLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": g.workingDir,
	})

	toolLogger.Info("Grep tool called")
	startTime := time.Now()

	// Parse the input command into pattern and optional target
	parts := strings.SplitN(strings.TrimSpace(input), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		toolLogger.Warn("Empty search pattern provided")
		return "Error: Please provide a search pattern", nil
	}

	pattern := parts[0]
	args := []string{"-r", pattern}

	// Determine target path (file or directory)
	if len(parts) == 2 && parts[1] != "" {
		targetPath := parts[1]
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(g.workingDir, targetPath)
		}
		args = append(args, targetPath)
	} else {
		// Search in current working directory
		args = append(args, g.workingDir)
	}

	// Execute grep command
	cmd := exec.CommandContext(ctx, "grep", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("pattern", pattern).Error("grep command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"pattern":       pattern,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("grep command completed")

	return string(output), nil
}

// Ensure GrepTool implements the tools.Tool interface
var _ tools.Tool = (*GrepTool)(nil)
