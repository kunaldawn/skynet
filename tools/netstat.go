package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

var netstatLogger = logrus.WithField("tool", "netstat")

type NetstatTool struct{}

func NewNetstatTool() *NetstatTool {
	netstatLogger.Debug("Initializing netstat tool")
	return &NetstatTool{}
}

func (n *NetstatTool) Description() string {
	return "Display network connections, routing tables, and network interface statistics. Usage: 'netstat' for all connections, 'netstat -l' for listening ports, 'netstat -r' for routing table, 'netstat -i' for interface stats."
}

func (n *NetstatTool) Name() string {
	return "netstat"
}

func (n *NetstatTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := netstatLogger.WithField("input", input)
	toolLogger.Info("Netstat tool called")
	startTime := time.Now()

	// Parse input options
	args := strings.Fields(strings.TrimSpace(input))
	if len(args) == 0 {
		// Default: show all connections
		args = []string{"-tuln"}
	}

	// Execute netstat command
	cmd := exec.CommandContext(ctx, "netstat", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		toolLogger.WithError(err).WithField("args", args).Error("netstat command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"args":          args,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("netstat command completed")

	return string(output), nil
}

var _ tools.Tool = (*NetstatTool)(nil)
