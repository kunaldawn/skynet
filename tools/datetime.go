package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var datetimeLogger = logrus.WithField("tool", "datetime")

type DateTimeTool struct{}

func NewDateTimeTool() *DateTimeTool {
	datetimeLogger.Debug("Initializing datetime tool")
	return &DateTimeTool{}
}

func (d *DateTimeTool) Description() string {
	return "Display current date and time. Usage: 'date' (current date/time), 'date -u' (UTC time), 'timedatectl' (system time info)."
}

func (d *DateTimeTool) Name() string {
	return "datetime"
}

func (d *DateTimeTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := datetimeLogger.WithField("input", input)
	toolLogger.Info("DateTime tool called")
	startTime := time.Now()

	command := strings.TrimSpace(input)
	if command == "" {
		command = "date"
	}

	var cmd *exec.Cmd
	parts := strings.Fields(command)

	switch parts[0] {
	case "date":
		if len(parts) > 1 {
			cmd = exec.CommandContext(ctx, "date", parts[1:]...)
		} else {
			cmd = exec.CommandContext(ctx, "date")
		}
	case "timedatectl":
		cmd = exec.CommandContext(ctx, "timedatectl")
	default:
		cmd = exec.CommandContext(ctx, "date")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithField("command", command).Error("DateTime command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("DateTime command completed")

	return string(output), nil
}

var _ tools.Tool = (*DateTimeTool)(nil)
