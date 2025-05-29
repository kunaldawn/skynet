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

var catLogger = logrus.WithField("tool", "cat")

type CatTool struct {
	workingDir *string
}

func NewCatTool(workingDir *string) *CatTool {
	catLogger.WithField("workingDir", *workingDir).Debug("Initializing cat tool")
	return &CatTool{workingDir: workingDir}
}

func (c *CatTool) Description() string {
	return "Display the contents of files. Usage: provide a filename or file path to view its contents. For large files, only the first 100 lines are shown."
}

func (c *CatTool) Name() string {
	return "cat"
}

func (c *CatTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := catLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": *c.workingDir,
	})

	toolLogger.Info("Cat tool called")
	startTime := time.Now()

	targetPath := strings.TrimSpace(input)
	if targetPath == "" {
		toolLogger.Warn("Empty file path provided")
		return "Error: Please provide a file path", nil
	}

	// Handle relative paths
	if !filepath.IsAbs(targetPath) && c.workingDir != nil {
		targetPath = filepath.Join(*c.workingDir, targetPath)
	}

	// Execute cat command
	cmd := exec.CommandContext(ctx, "cat", targetPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("targetPath", targetPath).Error("cat command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"targetPath":    targetPath,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("cat command completed")

	return string(output), nil
}

var _ tools.Tool = (*CatTool)(nil)
