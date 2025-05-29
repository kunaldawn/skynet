package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var lsLogger = logrus.WithField("tool", "ls")

type LsTool struct {
	workingDir string
}

func NewLsTool() *LsTool {
	wd, _ := os.Getwd()
	lsLogger.WithField("workingDir", wd).Debug("Initializing ls tool")
	return &LsTool{workingDir: wd}
}

func (l *LsTool) Description() string {
	return "List files and directories, or get information about a specific file. Use empty input or '.' for current directory, provide a directory path to list its contents, or provide a file path to get file information."
}

func (l *LsTool) Name() string {
	return "ls"
}

func (l *LsTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := lsLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": l.workingDir,
	})

	toolLogger.Info("Ls tool called")
	startTime := time.Now()

	targetPath := strings.TrimSpace(input)
	// Handle "None" input from agent as empty string
	if targetPath == "" || strings.ToLower(targetPath) == "none" {
		targetPath = "."
	}

	// If not absolute path, resolve relative to working directory
	if !filepath.IsAbs(targetPath) && l.workingDir != "" {
		targetPath = filepath.Join(l.workingDir, targetPath)
	}

	// Execute ls command
	cmd := exec.CommandContext(ctx, "ls", "-la", targetPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("targetPath", targetPath).Error("ls command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"targetPath":    targetPath,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("ls command completed")

	return string(output), nil
}

var _ tools.Tool = (*LsTool)(nil)
