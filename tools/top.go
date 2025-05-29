package tools

import (
	"context"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var topLogger = logrus.WithField("tool", "top")

type TopTool struct{}

func NewTopTool() *TopTool {
	topLogger.Debug("Initializing top tool")
	return &TopTool{}
}

func (t *TopTool) Description() string {
	return "Display system resource usage and running processes. Shows CPU, memory usage, and top processes. Runs as a one-time snapshot, not continuously."
}

func (t *TopTool) Name() string {
	return "top"
}

func (t *TopTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := topLogger.WithField("input", input)
	toolLogger.Info("Top tool called")
	startTime := time.Now()

	// Use top with batch mode for one-time output
	cmd := exec.CommandContext(ctx, "top", "-b", "-n", "1")
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).Error("top command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("top command completed")

	return string(output), nil
}

var _ tools.Tool = (*TopTool)(nil)
