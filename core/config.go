/*
Package core provides configuration management and logging initialization
for the Skynet Agent application.

This file handles:
- Loading configuration from environment variables with sensible defaults
- Structured logging setup with configurable levels and formats
- Performance and operational parameter management
- Session and memory management configuration

The configuration system follows the twelve-factor app methodology by
prioritizing environment variables for deployment flexibility while
providing reasonable defaults for development.
*/
package core

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds all configurable values for the Skynet Agent application.
// This structure centralizes all operational parameters including server settings,
// AI model configuration, performance tuning, and behavioral controls.
type Config struct {
	// Server configuration
	Port string // HTTP server port number (default: "8080")

	// LLM Provider configuration
	LLMProvider string // LLM provider to use: "ollama" or "gemini" (default: "ollama")

	// Ollama LLM configuration
	OllamaEndpoint string // Base URL for the Ollama API service (default: "http://localhost:11434")
	OllamaModel    string // Name of the Ollama model to use for inference (default: "qwen3")

	// Gemini LLM configuration
	GeminiAPIKey string // API key for Google Gemini (required when using gemini provider)
	GeminiModel  string // Name of the Gemini model to use for inference (default: "gemini-1.5-pro")

	// Agent execution configuration
	MaxIterations  int           // Maximum number of iterations for agent reasoning loops (default: 100)
	RequestTimeout time.Duration // Timeout for individual requests to prevent hanging (default: 300s)
	ContextLimit   int           // Maximum number of messages to include in conversation context (default: 10)

	// Memory store configuration for session management
	SessionMaxAge      time.Duration // How long to keep sessions in memory before expiring (default: 24h)
	CleanupInterval    time.Duration // How often to run cleanup of expired sessions (default: 1h)
	MaxSessionsPerUser int           // Maximum sessions allowed per user to prevent memory exhaustion (default: 50)

	// Logging and debugging configuration
	LogLevel          string // Minimum log level: debug, info, warn, error (default: "info")
	LogTruncateLength int    // Maximum length for log message truncation to prevent excessive output (default: 500)
	DebugMode         bool   // Enable debug mode for detailed internal logging (default: true)

	// Performance tuning parameters
	MaxConcurrentRequests int // Maximum number of concurrent requests to handle (default: 100)
}

// LoadConfig loads configuration from environment variables with sensible defaults.
// This function implements the configuration loading strategy by first setting
// reasonable defaults and then overriding them with environment variables if present.
// All environment variable parsing includes validation to ensure sensible values.
//
// Environment Variables:
//   - PORT: Server port (string)
//   - LLM_PROVIDER: LLM provider to use: "ollama" or "gemini" (string)
//   - OLLAMA_ENDPOINT: Ollama API endpoint URL (string)
//   - OLLAMA_MODEL: Model name for inference (string)
//   - GEMINI_API_KEY: Google Gemini API key (string)
//   - GEMINI_MODEL: Gemini model name for inference (string)
//   - MAX_ITERATIONS: Maximum agent iterations (integer)
//   - REQUEST_TIMEOUT: Request timeout in seconds (integer)
//   - CONTEXT_LIMIT: Maximum context messages (integer)
//   - SESSION_MAX_AGE_HOURS: Session expiry in hours (integer)
//   - CLEANUP_INTERVAL_MINUTES: Cleanup frequency in minutes (integer)
//   - MAX_SESSIONS_PER_USER: Maximum sessions per user (integer)
//   - LOG_LEVEL: Logging level (string)
//   - LOG_TRUNCATE_LENGTH: Log truncation length (integer)
//   - DEBUG_MODE: Enable debug mode (boolean: "true"/"1")
//   - MAX_CONCURRENT_REQUESTS: Concurrent request limit (integer)
func LoadConfig() *Config {
	// Initialize configuration with sensible defaults
	config := &Config{
		// Server defaults
		Port: "8080",

		// LLM Provider defaults
		LLMProvider: "gemini",

		// Ollama service defaults
		OllamaEndpoint: "http://localhost:11434",
		OllamaModel:    "qwen3",

		// Gemini service defaults
		GeminiAPIKey: "", // Must be provided via environment variable
		GeminiModel:  "gemini-2.0-flash",

		// Agent behavior defaults
		MaxIterations:  100,
		RequestTimeout: 300 * time.Second, // 5 minutes
		ContextLimit:   10,

		// Session management defaults
		SessionMaxAge:      24 * time.Hour, // 1 day
		CleanupInterval:    1 * time.Hour,  // 1 hour
		MaxSessionsPerUser: 50,

		// Logging defaults
		LogLevel:          "info",
		LogTruncateLength: 500,
		DebugMode:         true,

		// Performance defaults
		MaxConcurrentRequests: 100,
	}

	// Override defaults with environment variables if present

	// Server configuration
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}

	// LLM Provider configuration
	if provider := os.Getenv("LLM_PROVIDER"); provider != "" {
		if provider == "ollama" || provider == "gemini" {
			config.LLMProvider = provider
		}
	}

	// Ollama configuration
	if endpoint := os.Getenv("OLLAMA_ENDPOINT"); endpoint != "" {
		config.OllamaEndpoint = endpoint
	}

	if model := os.Getenv("OLLAMA_MODEL"); model != "" {
		config.OllamaModel = model
	}

	// Gemini configuration
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		config.GeminiAPIKey = apiKey
	}

	if model := os.Getenv("GEMINI_MODEL"); model != "" {
		config.GeminiModel = model
	}

	// Agent execution parameters with validation
	if maxIter := os.Getenv("MAX_ITERATIONS"); maxIter != "" {
		if val, err := strconv.Atoi(maxIter); err == nil && val > 0 {
			config.MaxIterations = val
		}
	}

	if timeout := os.Getenv("REQUEST_TIMEOUT"); timeout != "" {
		if val, err := strconv.Atoi(timeout); err == nil && val > 0 {
			config.RequestTimeout = time.Duration(val) * time.Second
		}
	}

	if contextLimit := os.Getenv("CONTEXT_LIMIT"); contextLimit != "" {
		if val, err := strconv.Atoi(contextLimit); err == nil && val > 0 {
			config.ContextLimit = val
		}
	}

	// Session management parameters with validation
	if sessionMaxAge := os.Getenv("SESSION_MAX_AGE_HOURS"); sessionMaxAge != "" {
		if val, err := strconv.Atoi(sessionMaxAge); err == nil && val > 0 {
			config.SessionMaxAge = time.Duration(val) * time.Hour
		}
	}

	if cleanupInterval := os.Getenv("CLEANUP_INTERVAL_MINUTES"); cleanupInterval != "" {
		if val, err := strconv.Atoi(cleanupInterval); err == nil && val > 0 {
			config.CleanupInterval = time.Duration(val) * time.Minute
		}
	}

	if maxSessions := os.Getenv("MAX_SESSIONS_PER_USER"); maxSessions != "" {
		if val, err := strconv.Atoi(maxSessions); err == nil && val > 0 {
			config.MaxSessionsPerUser = val
		}
	}

	// Logging configuration
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if truncateLen := os.Getenv("LOG_TRUNCATE_LENGTH"); truncateLen != "" {
		if val, err := strconv.Atoi(truncateLen); err == nil && val > 0 {
			config.LogTruncateLength = val
		}
	}

	// Debug mode parsing (accepts "true", "1", or case variations)
	if debug := os.Getenv("DEBUG_MODE"); debug != "" {
		config.DebugMode = strings.ToLower(debug) == "true" || debug == "1"
	}

	// Performance tuning
	if maxConcurrent := os.Getenv("MAX_CONCURRENT_REQUESTS"); maxConcurrent != "" {
		if val, err := strconv.Atoi(maxConcurrent); err == nil && val > 0 {
			config.MaxConcurrentRequests = val
		}
	}

	// Validate provider-specific configuration
	if config.LLMProvider == "gemini" && config.GeminiAPIKey == "" {
		// Note: We'll also validate this in the server initialization for better error messages
		// but this provides early validation during config loading
		config.LLMProvider = "ollama" // Fallback to ollama if Gemini key is missing
	}

	return config
}

// InitializeLogger configures and returns a structured logger based on the provided configuration.
// The logger uses JSON formatting for structured logging, which is ideal for production
// environments, log aggregation, and automated log processing.
//
// Features:
// - JSON formatted output for structured logging
// - Configurable log levels (debug, info, warn, error)
// - RFC3339 timestamp format for precise timing
// - Output to stdout for container-friendly logging
// - Configuration value logging for operational visibility
//
// Parameters:
//   - config: Configuration object containing logging preferences
//
// Returns:
//   - *logrus.Logger: Configured logger instance ready for use
func InitializeLogger(config *Config) *logrus.Logger {
	// Create new logger instance
	logger := logrus.New()

	// Configure JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339, // Use RFC3339 for ISO 8601 compatibility
	})

	// Set log level based on configuration with case-insensitive matching
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		// Default to info level if unrecognized level is specified
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set output to stdout for container/cloud environments
	// This allows log aggregation systems to capture logs properly
	logger.SetOutput(os.Stdout)

	// Log the loaded configuration for operational visibility
	// This helps with debugging configuration issues in production
	logger.WithFields(logrus.Fields{
		"llmProvider":           config.LLMProvider,
		"ollamaEndpoint":        config.OllamaEndpoint,
		"ollamaModel":           config.OllamaModel,
		"geminiModel":           config.GeminiModel,
		"maxIterations":         config.MaxIterations,
		"requestTimeout":        config.RequestTimeout,
		"contextLimit":          config.ContextLimit,
		"sessionMaxAge":         config.SessionMaxAge,
		"cleanupInterval":       config.CleanupInterval,
		"maxSessionsPerUser":    config.MaxSessionsPerUser,
		"logTruncateLength":     config.LogTruncateLength,
		"debugMode":             config.DebugMode,
		"maxConcurrentRequests": config.MaxConcurrentRequests,
	}).Info("Configuration loaded")

	return logger
}
