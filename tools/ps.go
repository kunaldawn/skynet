package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var psLogger = logrus.WithField("tool", "ps")

type PsTool struct{}

func NewPsTool() *PsTool {
	psLogger.Debug("Initializing ps tool")
	return &PsTool{}
}

func (p *PsTool) Description() string {
	return "Display running processes. Usage: 'ps' (show all user processes), 'ps aux' (detailed view), 'ps -ef' (full format), 'ps -u <user>' (processes by user), 'ps grep <pattern>' (filter processes by pattern)."
}

func (p *PsTool) Name() string {
	return "ps"
}

func (p *PsTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := psLogger.WithField("input", input)
	toolLogger.Info("PS tool called")
	startTime := time.Now()

	// Parse input
	args := strings.Fields(strings.TrimSpace(input))
	var cmd *exec.Cmd

	// Handle different ps options
	if len(args) == 0 || input == "" {
		// Default: show user processes
		cmd = exec.Command("ps", "-u", getUsername())
	} else if len(args) >= 2 && args[0] == "grep" {
		// Custom grep functionality
		pattern := strings.Join(args[1:], " ")
		psCmd := exec.Command("ps", "aux")
		grepCmd := exec.Command("grep", "-i", pattern)

		// Pipe ps output to grep
		pipe, err := psCmd.StdoutPipe()
		if err != nil {
			toolLogger.WithError(err).Error("Failed to create pipe")
			return "Error: Failed to create command pipe", nil
		}

		grepCmd.Stdin = pipe

		if err := psCmd.Start(); err != nil {
			toolLogger.WithError(err).Error("Failed to start ps command")
			return "Error: Failed to start ps command", nil
		}

		output, err := grepCmd.CombinedOutput()
		if err := psCmd.Wait(); err != nil {
			toolLogger.WithError(err).Error("PS command failed")
		}

		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			toolLogger.WithError(err).Error("Grep command failed")
			return string(output), nil
		}

		executionTime := time.Since(startTime)
		toolLogger.WithFields(logrus.Fields{
			"pattern":       pattern,
			"executionTime": executionTime,
		}).Info("Process grep completed")

		return string(output), nil
	} else {
		// Handle standard ps options directly
		cmd = exec.Command("ps", args...)
	}

	// Execute command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd = exec.CommandContext(cmdCtx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithField("output", string(output)).Error("PS command failed")

		if cmdCtx.Err() == context.DeadlineExceeded {
			return "Error: PS command timed out after 15 seconds", nil
		}

		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("PS command completed")

	return string(output), nil
}

// Helper functions
func getUsername() string {
	cmd := exec.Command("whoami")
	output, err := cmd.Output()
	if err != nil {
		return "root" // fallback
	}
	return strings.TrimSpace(string(output))
}

var _ tools.Tool = (*PsTool)(nil)
