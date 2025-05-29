package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var cdLogger = logrus.WithField("tool", "cd")

type CdTool struct {
	workingDir *string
}

func NewCdTool(workingDir *string) *CdTool {
	cdLogger.WithField("workingDir", *workingDir).Debug("Initializing cd tool")
	return &CdTool{workingDir: workingDir}
}

func (c *CdTool) Description() string {
	return "Change the current working directory. Usage: 'cd <path>' or 'cd' (go to home directory)."
}

func (c *CdTool) Name() string {
	return "cd"
}

func (c *CdTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := cdLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": *c.workingDir,
	})
	toolLogger.Info("CD tool called")
	startTime := time.Now()

	targetPath := strings.TrimSpace(input)

	// Handle empty input (go to home directory)
	if targetPath == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			targetPath = homeDir
		} else {
			targetPath = "/"
		}
	}

	// Resolve relative paths
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(*c.workingDir, targetPath)
	}

	// Clean the path
	targetPath = filepath.Clean(targetPath)

	// Check if directory exists
	info, err := os.Stat(targetPath)
	if err != nil {
		toolLogger.WithError(err).WithField("targetPath", targetPath).Error("Directory does not exist")
		return "Error: Directory does not exist: " + targetPath, nil
	}

	if !info.IsDir() {
		toolLogger.WithField("targetPath", targetPath).Error("Path is not a directory")
		return "Error: Path is not a directory: " + targetPath, nil
	}

	// Update working directory
	*c.workingDir = targetPath

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"newWorkingDir": targetPath,
		"executionTime": executionTime,
	}).Info("Directory changed successfully")

	return "Changed directory to: " + targetPath, nil
}

var _ tools.Tool = (*CdTool)(nil)
