package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var apkLogger = logrus.WithField("tool", "apk")

type ApkTool struct{}

func NewApkTool() *ApkTool {
	apkLogger.Debug("Initializing APK tool")
	return &ApkTool{}
}

func (a *ApkTool) Description() string {
	return "Alpine Package Keeper (APK) package management. Supports all APK commands including: update, search <package>, info <package>, list, add <package>, del <package>, upgrade, fix, cache clean/sync/download, version, policy <package>, etc. Full APK functionality is available."
}

func (a *ApkTool) Name() string {
	return "apk"
}

func (a *ApkTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := apkLogger.WithField("input", input)
	toolLogger.Info("APK tool called")
	startTime := time.Now()

	// Parse input command
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		toolLogger.Warn("Empty APK command provided")
		return "Error: Please provide an APK command. All APK commands are supported.", nil
	}

	// Execute command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "apk", parts...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithFields(logrus.Fields{
			"command": parts[0],
			"output":  string(output),
		}).Error("APK command failed")

		if cmdCtx.Err() == context.DeadlineExceeded {
			return "Error: APK command timed out after 60 seconds", nil
		}

		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       parts[0],
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("APK command completed")

	return string(output), nil
}

var _ tools.Tool = (*ApkTool)(nil)
