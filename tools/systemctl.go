package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var systemctlLogger = logrus.WithField("tool", "systemctl")

type SystemctlTool struct{}

func NewSystemctlTool() *SystemctlTool {
	systemctlLogger.Debug("Initializing systemctl tool")
	return &SystemctlTool{}
}

func (s *SystemctlTool) Description() string {
	return "Control and query systemd services and system state. Supports all systemctl commands including: status <service>, list, failed, active, enabled, logs <service>, show <service>, start <service>, stop <service>, restart <service>, reload <service>, enable <service>, disable <service>, mask <service>, unmask <service>, etc. Full systemctl functionality is available."
}

func (s *SystemctlTool) Name() string {
	return "systemctl"
}

func (s *SystemctlTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := systemctlLogger.WithField("input", input)
	toolLogger.Info("Systemctl tool called")
	startTime := time.Now()

	// Parse input command
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		toolLogger.Warn("Empty systemctl command provided")
		return "Error: Please provide a systemctl command. All systemctl commands are supported.", nil
	}

	command := strings.ToLower(parts[0])

	// Execute command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "systemctl", parts...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithFields(logrus.Fields{
			"command": command,
			"output":  string(output),
		}).Error("Systemctl command failed")

		if cmdCtx.Err() == context.DeadlineExceeded {
			return "Error: Systemctl command timed out after 30 seconds", nil
		}

		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Systemctl command completed")

	return string(output), nil
}

var _ tools.Tool = (*SystemctlTool)(nil)
