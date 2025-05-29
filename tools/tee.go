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

var teeLogger = logrus.WithField("tool", "tee")

type TeeTool struct {
	workingDir *string
}

func NewTeeTool(workingDir *string) *TeeTool {
	teeLogger.WithField("workingDir", *workingDir).Debug("Initializing tee tool")
	return &TeeTool{workingDir: workingDir}
}

func (t *TeeTool) Description() string {
	return "Write input to both stdout and file(s). Usage: 'tee <file> <input>' (write to file and display), 'tee -a <file> <input>' (append to file and display)."
}

func (t *TeeTool) Name() string {
	return "tee"
}

func (t *TeeTool) Call(ctx context.Context, input string) (string, error) {
	toolLogger := teeLogger.WithFields(logrus.Fields{
		"input":      input,
		"workingDir": *t.workingDir,
	})
	toolLogger.Info("Tee tool called")
	startTime := time.Now()

	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) < 2 {
		toolLogger.Warn("Insufficient arguments provided")
		return "Error: Please provide a filename and input text", nil
	}

	var args []string
	var filename string
	var content string

	// Check for append flag
	if parts[0] == "-a" {
		if len(parts) < 3 {
			return "Error: Please provide a filename and input text after -a flag", nil
		}
		args = append(args, "-a")
		filename = parts[1]
		content = strings.Join(parts[2:], " ")
	} else {
		filename = parts[0]
		content = strings.Join(parts[1:], " ")
	}

	// Handle relative paths
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(*t.workingDir, filename)
	}

	args = append(args, filename)

	// Execute tee command
	cmd := exec.CommandContext(ctx, "tee", args...)
	cmd.Dir = *t.workingDir

	// Provide input to tee
	cmd.Stdin = strings.NewReader(content)

	output, err := cmd.CombinedOutput()
	if err != nil {
		toolLogger.WithError(err).WithField("filename", filename).Error("Tee command failed")
		return string(output), nil
	}

	executionTime := time.Since(startTime)
	toolLogger.WithFields(logrus.Fields{
		"filename":      filename,
		"executionTime": executionTime,
		"outputLength":  len(string(output)),
	}).Info("Tee command completed")

	return string(output), nil
}

var _ tools.Tool = (*TeeTool)(nil)
