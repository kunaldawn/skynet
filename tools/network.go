/*
Package tools provides network diagnostics and configuration capabilities for the Skynet Agent.

This file implements the NetworkTool, which provides comprehensive network functionality
including interface management, routing diagnostics, DNS configuration, connectivity testing,
and network monitoring. The tool acts as a bridge between the agent and various network
utilities, enabling full network diagnostic capabilities through the agent interface.

Supported operations:
- Interface Management: interfaces (show network interfaces and statistics)
- Routing: routes (display routing table information)
- DNS: dns (show DNS configuration and hosts file)
- Connectivity: ping (test network connectivity to hosts)
- ARP: arp (display ARP table entries)
- Connections: connections (show active network connections and listening ports)
- All standard network commands: ifconfig, ip, netstat, ss, route, traceroute, nslookup, dig, wget, curl, etc.

The tool provides enhanced formatting for common diagnostic operations while
supporting the full range of network utilities for advanced operations.
*/
package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/tools"
)

// networkLogger provides structured logging for all network operations
// with a consistent tool identifier for easy filtering and monitoring
var networkLogger = logrus.WithField("tool", "network")

// NetworkTool provides comprehensive network diagnostics and configuration capabilities.
// It wraps various network utilities to provide agent-accessible network operations with
// enhanced formatting, error handling, and logging for operational monitoring.
type NetworkTool struct{}

// NewNetworkTool creates a new instance of the network diagnostics tool.
// The tool requires standard network utilities to be available in the system PATH.
//
// Returns:
//   - *NetworkTool: Configured network tool ready for use
func NewNetworkTool() *NetworkTool {
	networkLogger.Debug("Initializing network tool")
	return &NetworkTool{}
}

// Description returns a comprehensive description of the network tool's capabilities.
// This description is used by the agent framework to understand what network
// operations are available and how to invoke them properly.
//
// Returns:
//   - string: Detailed description of all supported network operations
func (n *NetworkTool) Description() string {
	return "Network connectivity and diagnostics. Usage: 'ping <host>' (ping host), 'wget <url>' (download), 'curl <url>' (HTTP request), 'dig <domain>' (DNS lookup), 'traceroute <host>' (trace route), 'whois <domain>' (domain info), 'nslookup <domain>' (DNS lookup)."
}

// Name returns the identifier for this tool.
// This name is used by the agent framework for tool selection and invocation.
//
// Returns:
//   - string: The tool's identifier ("network")
func (n *NetworkTool) Name() string {
	return "network"
}

// Call executes a network command based on the provided input.
// This is the main entry point for all network operations. The method parses
// the input command, validates the operation, and executes the requested
// network diagnostic or configuration task with proper error handling,
// timeout management, and result formatting.
//
// The method provides enhanced formatting for common diagnostic operations like
// interfaces, routes, dns, ping, arp, and connections, while supporting the full
// range of network utilities for advanced operations.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - input: Network command string (e.g., "ping google.com", "interfaces")
//
// Returns:
//   - string: Formatted result of the network operation or error message
//   - error: Always nil (errors are returned as string messages)
func (n *NetworkTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := networkLogger.WithField("input", input)
	toolLogger.Info("Network tool called")
	startTime := time.Now()

	// Parse input command
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		toolLogger.Warn("Empty network command provided")
		return "Error: Please provide a network command", nil
	}

	command := strings.ToLower(parts[0])

	// Execute the appropriate network command
	var cmd *exec.Cmd
	switch command {
	case "ping":
		if len(parts) < 2 {
			return "Error: Please specify a host to ping", nil
		}
		host := parts[1]
		cmd = exec.CommandContext(ctx, "ping", "-c", "4", host)

	case "wget":
		if len(parts) < 2 {
			return "Error: Please specify a URL to download", nil
		}
		url := parts[1]
		cmd = exec.CommandContext(ctx, "wget", "-q", "-O", "-", url)

	case "curl":
		if len(parts) < 2 {
			return "Error: Please specify a URL for curl", nil
		}
		url := parts[1]
		cmd = exec.CommandContext(ctx, "curl", "-s", url)

	case "dig":
		if len(parts) < 2 {
			return "Error: Please specify a domain for dig", nil
		}
		domain := parts[1]
		cmd = exec.CommandContext(ctx, "dig", domain)

	case "traceroute":
		if len(parts) < 2 {
			return "Error: Please specify a host for traceroute", nil
		}
		host := parts[1]
		cmd = exec.CommandContext(ctx, "traceroute", host)

	case "whois":
		if len(parts) < 2 {
			return "Error: Please specify a domain for whois", nil
		}
		domain := parts[1]
		cmd = exec.CommandContext(ctx, "whois", domain)

	case "nslookup":
		if len(parts) < 2 {
			return "Error: Please specify a domain for nslookup", nil
		}
		domain := parts[1]
		cmd = exec.CommandContext(ctx, "nslookup", domain)

	default:
		return "Error: Unsupported network command. Supported: ping, wget, curl, dig, traceroute, whois, nslookup", nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithField("command", command).Error("Network command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"command":       command,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Network command completed")

	return string(output), nil
}

// Ensure NetworkTool implements the tools.Tool interface
var _ tools.Tool = (*NetworkTool)(nil)
