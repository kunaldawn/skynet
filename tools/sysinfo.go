package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var sysinfoLogger = logrus.WithField("tool", "sysinfo")

type SysInfoTool struct{}

func NewSysInfoTool() *SysInfoTool {
	sysinfoLogger.Debug("Initializing sysinfo tool")
	return &SysInfoTool{}
}

func (s *SysInfoTool) Description() string {
	return "Display system information. Usage: 'uname' (system info), 'uptime' (uptime), 'free' (memory), 'df' (disk usage), 'lscpu' (CPU info), 'lsblk' (block devices), 'mount' (mounted filesystems)."
}

func (s *SysInfoTool) Name() string {
	return "sysinfo"
}

func (s *SysInfoTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := sysinfoLogger.WithField("input", input)
	toolLogger.Info("Sysinfo tool called")
	startTime := time.Now()

	// Parse input command
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		parts = []string{"all"} // Default to showing all info
	}

	command := strings.ToLower(parts[0])

	var cmd *exec.Cmd
	switch command {
	case "all":
		// Show basic system overview
		cmd = exec.CommandContext(ctx, "uname", "-a")

	case "uname":
		cmd = exec.CommandContext(ctx, "uname", "-a")

	case "uptime":
		cmd = exec.CommandContext(ctx, "uptime")

	case "free":
		cmd = exec.CommandContext(ctx, "free", "-h")

	case "df":
		cmd = exec.CommandContext(ctx, "df", "-h")

	case "lscpu":
		cmd = exec.CommandContext(ctx, "lscpu")

	case "lsblk":
		cmd = exec.CommandContext(ctx, "lsblk")

	case "mount":
		cmd = exec.CommandContext(ctx, "mount")

	default:
		return "Error: Unsupported sysinfo command. Supported: all, uname, uptime, free, df, lscpu, lsblk, mount", nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithField("command", command).Error("Sysinfo command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Sysinfo command completed")

	return string(output), nil
}

var _ tools.Tool = (*SysInfoTool)(nil)
