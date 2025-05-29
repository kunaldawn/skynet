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

var statLogger = logrus.WithField("tool", "stat")

type StatTool struct {
	workingDir *string
}

func NewStatTool(workingDir *string) *StatTool {
	statLogger.WithField("workingDir", *workingDir).Debug("Initializing stat tool")
	return &StatTool{workingDir: workingDir}
}

func (s *StatTool) Description() string {
	return "Display detailed file or directory information including size, permissions, timestamps, and more. Usage: provide a file or directory path."
}

func (s *StatTool) Name() string {
	return "stat"
}

func (s *StatTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := statLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": *s.workingDir,
	})

	toolLogger.Info("Stat tool called")
	startTime := time.Now()

	targetPath := strings.TrimSpace(input)
	if targetPath == "" {
		toolLogger.Warn("Empty path provided")
		return "Error: Please provide a file or directory path", nil
	}

	// Handle relative paths
	if !filepath.IsAbs(targetPath) && s.workingDir != nil {
		targetPath = filepath.Join(*s.workingDir, targetPath)
	}

	// Execute stat command
	cmd := exec.CommandContext(ctx, "stat", targetPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("targetPath", targetPath).Error("stat command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"targetPath":    targetPath,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("stat command completed")

	return string(output), nil
}

var _ tools.Tool = (*StatTool)(nil)
