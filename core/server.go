package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	localtools "skynet/tools"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/tools"
)

type Server struct {
	executor      *agents.Executor
	toolsList     []tools.Tool
	memoryStore   *MemoryStore
	cancelManager *CancelManager
	config        *Config
	logger        *logrus.Logger
}

// NewServer creates a new server instance with all dependencies initialized
func NewServer(config *Config, logger *logrus.Logger) (*Server, error) {
	logger.Info("Starting server initialization")

	workingDir, err := os.Getwd()
	if err != nil {
		logger.WithError(err).Error("Failed to get working directory")
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	logger.WithField("workingDir", workingDir).Info("Working directory set")

	// Initialize memory store
	memoryStore := NewMemoryStore(config.SessionMaxAge, config.CleanupInterval, logger)
	logger.WithField("sessionMaxAge", config.SessionMaxAge).Info("Memory store initialized with configurable session expiry")

	// Initialize LLM based on configured provider
	var llm llms.Model

	switch config.LLMProvider {
	case "gemini":
		logger.WithField("provider", "gemini").Info("Initializing Gemini LLM")

		// Validate API key for Gemini
		if config.GeminiAPIKey == "" {
			logger.Error("Gemini API key is required when using gemini provider")
			return nil, fmt.Errorf("gemini API key is required when using gemini provider. Set GEMINI_API_KEY environment variable")
		}

		modelName := config.GeminiModel
		if modelName == "" {
			modelName = "gemini-1.5-pro"
		}
		logger.WithField("model", modelName).Info("Using Gemini model")

		logger.Debug("Initializing Gemini LLM connection")
		llm, err = googleai.New(
			context.Background(),
			googleai.WithAPIKey(config.GeminiAPIKey),
			googleai.WithDefaultModel(modelName),
		)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"provider": "gemini",
				"model":    modelName,
			}).Error("Failed to initialize Gemini LLM")
			return nil, fmt.Errorf("failed to initialize Gemini LLM: %w", err)
		}
		logger.Info("Gemini LLM initialized successfully")

	case "ollama":
		fallthrough
	default:
		logger.WithField("provider", "ollama").Info("Initializing Ollama LLM")

		ollamaEndpoint := config.OllamaEndpoint
		if ollamaEndpoint == "" {
			ollamaEndpoint = "http://localhost:11434"
		}
		logger.WithField("endpoint", ollamaEndpoint).Info("Using Ollama endpoint")

		modelName := config.OllamaModel
		if modelName == "" {
			modelName = "qwen3"
		}
		logger.WithField("model", modelName).Info("Using Ollama model")

		logger.Debug("Initializing Ollama LLM connection")
		llm, err = ollama.New(
			ollama.WithServerURL(ollamaEndpoint),
			ollama.WithModel(modelName),
		)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"endpoint": ollamaEndpoint,
				"model":    modelName,
			}).Error("Failed to initialize Ollama LLM")
			return nil, fmt.Errorf("failed to initialize Ollama LLM: %w", err)
		}
		logger.Info("Ollama LLM initialized successfully")
	}

	// Wrap the LLM with the cleaning wrapper to handle think tags
	cleanedLLM := NewCleaningLLMWrapper(llm, config, logger)
	logger.Info("LLM wrapped with response cleaning functionality")

	// Initialize tools slice
	logger.Debug("Initializing tools")
	toolsList := []tools.Tool{
		localtools.NewDateTimeTool(),
		localtools.NewLsTool(),
		localtools.NewCdTool(&workingDir),
		localtools.NewTopTool(),
		localtools.NewGrepTool(&workingDir),
		localtools.NewStatTool(&workingDir),
		localtools.NewCatTool(&workingDir),
		localtools.NewFileTool(&workingDir),
		localtools.NewShellTool(&workingDir),
		localtools.NewTeeTool(&workingDir),
		localtools.NewDockerTool(),
		localtools.NewPsTool(),
		localtools.NewNetstatTool(),
		localtools.NewSysInfoTool(),
		localtools.NewSystemctlTool(),
		localtools.NewApkTool(),
	}
	logger.WithField("toolsCount", len(toolsList)).Info("Tools initialized")

	// Create agent executor with ZeroShotReact pattern for better tool handling
	logger.Debug("Creating agent executor with ZeroShotReact pattern")

	// Create a general verbose callback handler for the executor
	generalCallbackHandler := NewVerboseCallbackHandler(logger.WithField("component", "agent"), config)

	// Create custom optimized prompt for minimal tool usage
	customPrompt := CreateOptimizedPrompt(toolsList)

	executor, err := agents.Initialize(
		cleanedLLM,
		toolsList,
		agents.ZeroShotReactDescription,
		agents.WithPrompt(customPrompt), // Use custom optimized prompt
		agents.WithMaxIterations(config.MaxIterations),      // Use configured max iterations
		agents.WithReturnIntermediateSteps(),                // Enable intermediate steps for debugging
		agents.WithCallbacksHandler(generalCallbackHandler), // Add verbose logging
	)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize agent executor")
		return nil, fmt.Errorf("failed to initialize agent executor: %w", err)
	}

	logger.Info("Server initialization completed successfully")
	return &Server{
		executor:      executor,
		toolsList:     toolsList,
		memoryStore:   memoryStore,
		cancelManager: NewCancelManager(),
		config:        config,
		logger:        logger,
	}, nil
}

func (s *Server) handleChat(c echo.Context) error {
	requestID := c.Request().Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	requestLogger := s.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"endpoint":  "/chat",
		"method":    "POST",
		"clientIP":  c.RealIP(),
	})

	requestLogger.Info("Received chat request")

	var req ChatRequest
	if err := c.Bind(&req); err != nil {
		requestLogger.WithError(err).Error("Failed to parse request body")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get or create chat session
	session := s.memoryStore.GetOrCreateSession(req.SessionID)

	requestLogger.WithFields(logrus.Fields{
		"sessionID":     session.ID,
		"messageLength": len(req.Message),
		"message":       req.Message,
		"messageCount":  len(session.Messages),
	}).Debug("Chat request details with session info")

	// Add user message to session memory
	session.AddMessage("user", req.Message)

	// Create context with timeout to prevent long-running requests
	ctx, cancel := context.WithTimeout(context.Background(), s.config.RequestTimeout)
	defer cancel()

	startTime := time.Now()

	requestLogger.WithField("sessionID", session.ID).Info("Starting agent execution with memory context")

	// Build message with conversation context
	var messageWithContext string
	if len(session.Messages) > 1 { // More than just the current message
		// Include recent conversation history
		conversationContext := session.GetConversationContext(s.config.ContextLimit)
		messageWithContext = conversationContext + "Human: " + req.Message

		requestLogger.WithFields(logrus.Fields{
			"sessionID":      session.ID,
			"includeContext": true,
			"contextLength":  len(conversationContext),
		}).Debug("Including conversation context in request")
	} else {
		messageWithContext = req.Message
		requestLogger.WithField("sessionID", session.ID).Debug("No previous context, using message as-is")
	}

	// Use chains.Run directly with the executor
	result, err := chains.Run(ctx, s.executor, messageWithContext)
	executionTime := time.Since(startTime)

	if err != nil {
		// Log the error for debugging
		requestLogger.WithError(err).WithFields(logrus.Fields{
			"sessionID":     session.ID,
			"executionTime": executionTime,
			"message":       req.Message,
		}).Error("Agent execution failed")

		// Provide a more helpful error message to the user
		errorMsg := s.getErrorMessage(err)

		// Don't add error responses to memory
		requestLogger.WithFields(logrus.Fields{
			"sessionID":     session.ID,
			"errorType":     "execution_error",
			"userMessage":   errorMsg,
			"executionTime": executionTime,
		}).Warn("Returning error response to user")

		return c.JSON(http.StatusOK, ChatResponse{
			Response:  errorMsg,
			SessionID: session.ID,
		})
	}

	// Add assistant response to session memory
	session.AddMessage("assistant", result)

	requestLogger.WithFields(logrus.Fields{
		"sessionID":      session.ID,
		"executionTime":  executionTime,
		"responseLength": len(result),
		"response":       result,
		"messageCount":   len(session.Messages),
	}).Info("Agent execution completed successfully with memory updated")

	return c.JSON(http.StatusOK, ChatResponse{
		Response:  result,
		SessionID: session.ID,
	})
}

func (s *Server) handleStreamChat(c echo.Context) error {
	requestID := c.Request().Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("stream_req_%d", time.Now().UnixNano())
	}

	requestLogger := s.logger.WithFields(logrus.Fields{
		"requestId": requestID,
		"endpoint":  "/chat/stream",
		"method":    "POST",
		"clientIP":  c.RealIP(),
	})

	requestLogger.Info("Received streaming chat request")

	var req ChatRequest
	if err := c.Bind(&req); err != nil {
		requestLogger.WithError(err).Error("Failed to parse streaming request body")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Get or create chat session
	session := s.memoryStore.GetOrCreateSession(req.SessionID)

	requestLogger.WithFields(logrus.Fields{
		"sessionID":     session.ID,
		"messageLength": len(req.Message),
		"message":       req.Message,
		"messageCount":  len(session.Messages),
	}).Debug("Streaming chat request details with session info")

	// Add user message to session memory
	session.AddMessage("user", req.Message)

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	// Send session ID to client first
	s.sendStreamMessage(c, StreamMessage{
		Type:    "session",
		Content: session.ID,
	})

	// Generate execution ID for tracking and cancellation
	executionID := fmt.Sprintf("exec_%d", time.Now().UnixNano())

	// Send execution ID to client for stop functionality
	s.sendStreamMessage(c, StreamMessage{
		Type:    "execution_started",
		Content: executionID,
	})

	// Create context with timeout to prevent long-running requests
	ctx, cancel := context.WithTimeout(context.Background(), s.config.RequestTimeout)
	defer func() {
		// Always remove execution when done
		s.cancelManager.RemoveExecution(executionID)
		cancel()
	}()

	// Register execution for cancellation
	s.cancelManager.AddExecution(executionID, cancel)

	startTime := time.Now()

	requestLogger.WithFields(logrus.Fields{
		"sessionID":   session.ID,
		"executionID": executionID,
	}).Info("Starting streaming execution with memory context")

	// Send initial thinking message
	s.sendStreamMessage(c, StreamMessage{
		Type:    "thinking",
		Content: "Processing your request...",
	})

	// Build message with conversation context
	var messageWithContext string
	if len(session.Messages) > 1 { // More than just the current message
		// Include recent conversation history
		conversationContext := session.GetConversationContext(s.config.ContextLimit)
		messageWithContext = conversationContext + "Human: " + req.Message

		requestLogger.WithFields(logrus.Fields{
			"sessionID":      session.ID,
			"includeContext": true,
			"contextLength":  len(conversationContext),
		}).Debug("Including conversation context in streaming request")
	} else {
		messageWithContext = req.Message
		requestLogger.WithField("sessionID", session.ID).Debug("No previous context for streaming, using message as-is")
	}

	// Create a custom chain wrapper to capture intermediate steps
	result, err := s.executeWithStreaming(ctx, messageWithContext, s.config.DebugMode, c, requestLogger)
	executionTime := time.Since(startTime)

	if err != nil {
		requestLogger.WithError(err).WithFields(logrus.Fields{
			"sessionID":     session.ID,
			"executionID":   executionID,
			"executionTime": executionTime,
			"message":       req.Message,
		}).Error("Streaming agent execution failed")

		// Check if it was cancelled
		if ctx.Err() == context.Canceled {
			s.sendStreamMessage(c, StreamMessage{
				Type:    "stopped",
				Content: "Agent execution was stopped",
			})
			return nil
		}

		// Send appropriate error message based on error type
		errorMsg := s.getErrorMessage(err)

		// Don't add error responses to memory
		requestLogger.WithFields(logrus.Fields{
			"sessionID":     session.ID,
			"errorType":     "streaming_execution_error",
			"userMessage":   errorMsg,
			"executionTime": executionTime,
		}).Warn("Sending error message to streaming client")

		s.sendStreamMessage(c, StreamMessage{
			Type:    "error",
			Content: errorMsg,
		})
		return nil
	}

	// Add assistant response to session memory
	session.AddMessage("assistant", result)

	requestLogger.WithFields(logrus.Fields{
		"sessionID":      session.ID,
		"executionID":    executionID,
		"executionTime":  executionTime,
		"responseLength": len(result),
		"response":       result,
		"messageCount":   len(session.Messages),
	}).Info("Streaming execution completed successfully with memory updated")

	// Send final response
	s.sendStreamMessage(c, StreamMessage{
		Type:     "response",
		Content:  result,
		Complete: true,
	})

	return nil
}

func (s *Server) executeWithStreaming(ctx context.Context, message string, debug bool, c echo.Context, requestLogger *logrus.Entry) (string, error) {
	requestLogger.WithField("debugMode", debug).Debug("Starting streaming execution")

	// Send thinking message
	s.sendStreamMessage(c, StreamMessage{
		Type:    "thinking",
		Content: "Processing your request...",
		Debug:   debug,
	})

	requestLogger.Info("Starting chain execution")
	chainStartTime := time.Now()

	var result string
	var err error

	// Wrap execution in a recovery function to handle potential panics
	func() {
		defer func() {
			if r := recover(); r != nil {
				requestLogger.WithField("panic", r).Error("Panic occurred during execution")
				err = fmt.Errorf("execution failed due to internal error: %v", r)
			}
		}()

		if debug {
			// Create a custom executor with streaming callback handler for debug mode
			requestLogger.Info("Creating debug-enabled executor with streaming callbacks")

			// Get the working directory for tools
			workingDir, dirErr := os.Getwd()
			if dirErr != nil {
				requestLogger.WithError(dirErr).Error("Failed to get working directory for debug executor")
				err = fmt.Errorf("failed to get working directory: %w", dirErr)
				return
			}

			// Initialize LLM based on configured provider
			var llm llms.Model

			switch s.config.LLMProvider {
			case "gemini":
				requestLogger.WithField("provider", "gemini").Info("Initializing Gemini LLM")

				// Validate API key for Gemini
				if s.config.GeminiAPIKey == "" {
					requestLogger.Error("Gemini API key is required when using gemini provider")
					return
				}

				modelName := s.config.GeminiModel
				if modelName == "" {
					modelName = "gemini-1.5-pro"
				}
				requestLogger.WithField("model", modelName).Info("Using Gemini model")

				requestLogger.Debug("Initializing Gemini LLM connection")
				llm, err = googleai.New(
					context.Background(),
					googleai.WithAPIKey(s.config.GeminiAPIKey),
					googleai.WithDefaultModel(modelName),
				)
				if err != nil {
					requestLogger.WithError(err).WithFields(logrus.Fields{
						"provider": "gemini",
						"model":    modelName,
					}).Error("Failed to initialize Gemini LLM")
					return
				}
				requestLogger.Info("Gemini LLM initialized successfully")

			case "ollama":
				fallthrough
			default:
				requestLogger.WithField("provider", "ollama").Info("Initializing Ollama LLM")

				ollamaEndpoint := s.config.OllamaEndpoint
				if ollamaEndpoint == "" {
					ollamaEndpoint = "http://localhost:11434"
				}
				requestLogger.WithField("endpoint", ollamaEndpoint).Info("Using Ollama endpoint")

				modelName := s.config.OllamaModel
				if modelName == "" {
					modelName = "qwen3"
				}
				requestLogger.WithField("model", modelName).Info("Using Ollama model")

				requestLogger.Debug("Initializing Ollama LLM connection")
				llm, err = ollama.New(
					ollama.WithServerURL(ollamaEndpoint),
					ollama.WithModel(modelName),
				)
				if err != nil {
					requestLogger.WithError(err).WithFields(logrus.Fields{
						"endpoint": ollamaEndpoint,
						"model":    modelName,
					}).Error("Failed to initialize Ollama LLM")
					return
				}
				requestLogger.Info("Ollama LLM initialized successfully")
			}

			// Wrap the debug LLM with cleaning wrapper too
			cleanedDebugLLM := NewCleaningLLMWrapper(llm, s.config, s.logger)

			// Create streaming callback handler
			streamingHandler := NewStreamingCallbackHandler(
				requestLogger.WithField("component", "debug_agent"),
				s.config,
				func(msg StreamMessage) {
					s.sendStreamMessage(c, msg)
				},
			)

			// Initialize tools for debug executor
			debugToolsList := []tools.Tool{
				localtools.NewDateTimeTool(),
				localtools.NewLsTool(),
				localtools.NewCdTool(&workingDir),
				localtools.NewTopTool(),
				localtools.NewGrepTool(&workingDir),
				localtools.NewStatTool(&workingDir),
				localtools.NewCatTool(&workingDir),
				localtools.NewFileTool(&workingDir),
				localtools.NewShellTool(&workingDir),
				localtools.NewTeeTool(&workingDir),
				localtools.NewDockerTool(),
				localtools.NewPsTool(),
				localtools.NewNetstatTool(),
				localtools.NewSysInfoTool(),
				localtools.NewSystemctlTool(),
				localtools.NewApkTool(),
			}

			// Create debug executor with streaming callbacks
			customPrompt := CreateOptimizedPrompt(debugToolsList)

			debugExecutor, execErr := agents.Initialize(
				cleanedDebugLLM, // Use cleaned LLM wrapper
				debugToolsList,
				agents.ZeroShotReactDescription,
				agents.WithPrompt(customPrompt),                  // Use same optimized prompt as main executor
				agents.WithMaxIterations(s.config.MaxIterations), // Reduced to match main executor
				agents.WithReturnIntermediateSteps(),
				agents.WithCallbacksHandler(streamingHandler),
			)
			if execErr != nil {
				requestLogger.WithError(execErr).Error("Failed to initialize debug executor")
				err = fmt.Errorf("failed to initialize debug executor: %w", execErr)
				return
			}

			// Use the debug executor
			result, err = chains.Run(ctx, debugExecutor, message)
		} else {
			// Use the standard executor for non-debug mode
			result, err = chains.Run(ctx, s.executor, message)
		}

		// Handle specific parsing errors
		if err != nil && strings.Contains(err.Error(), "unable to parse agent output") {
			requestLogger.WithError(err).Error("Agent output parsing failed - likely due to malformed response")

			// Try to extract a meaningful response from the error message
			if strings.Contains(err.Error(), "unable to parse agent output: ") {
				// Extract the actual response that failed to parse
				errorParts := strings.SplitN(err.Error(), "unable to parse agent output: ", 2)
				if len(errorParts) > 1 {
					rawResponse := errorParts[1]
					// Try to clean and extract a meaningful response
					cleaned := s.cleanAgentResponse(rawResponse)
					if cleaned != "" {
						// Check if the cleaned response now follows proper format
						if strings.Contains(cleaned, "Final Answer:") {
							requestLogger.Info("Successfully recovered response from parsing error")
							// Extract just the final answer part
							finalAnswerRegex := regexp.MustCompile(`(?s)Final Answer:\s*(.*)`)
							matches := finalAnswerRegex.FindStringSubmatch(cleaned)
							if len(matches) > 1 {
								result = strings.TrimSpace(matches[1])
							} else {
								result = cleaned
							}
							err = nil
							return
						}
					}
				}
			}

			// If we can't recover, provide a helpful error
			err = fmt.Errorf("the agent generated a malformed response that couldn't be parsed")
		}
	}()

	if err != nil {
		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			requestLogger.Warn("Chain execution timed out")
			return "", fmt.Errorf("request timed out after 300 seconds")
		}

		chainExecutionTime := time.Since(chainStartTime)
		requestLogger.WithError(err).WithField("chainExecutionTime", chainExecutionTime).Error("Chain execution failed in streaming")
		return "", err
	}

	chainExecutionTime := time.Since(chainStartTime)

	requestLogger.WithFields(logrus.Fields{
		"chainExecutionTime": chainExecutionTime,
		"resultLength":       len(result),
	}).Info("Chain execution completed successfully")

	s.sendStreamMessage(c, StreamMessage{
		Type:    "thinking",
		Content: "Formatting response...",
		Debug:   debug,
	})

	return result, nil
}

func (s *Server) sendStreamMessage(c echo.Context, msg StreamMessage) {
	data, _ := json.Marshal(msg)
	fmt.Fprintf(c.Response(), "data: %s\n\n", string(data))
	c.Response().Flush()
}

func (s *Server) getErrorMessage(err error) string {
	errorMsg := "I encountered an error processing your request. "
	if strings.Contains(err.Error(), "unable to parse") {
		errorMsg += "The agent had trouble interpreting the tool output. Please try rephrasing your request."
	} else if strings.Contains(err.Error(), "max iterations") {
		errorMsg += "The request was too complex and required too many steps to complete. Please try breaking it down into simpler requests or be more specific about what you need."
	} else if strings.Contains(err.Error(), "context") {
		errorMsg += "The request timed out. Please try a simpler request."
	} else {
		errorMsg += "Please try again or contact support if the issue persists."
	}
	return errorMsg
}

func (s *Server) cleanAgentResponse(response string) string {
	// Create a temporary cleaning LLM wrapper to use the cleaning functionality
	tempWrapper := NewCleaningLLMWrapper(nil, s.config, s.logger)
	return tempWrapper.CleanAgentResponse(response)
}

func (s *Server) handleStatus(c echo.Context) error {
	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint": "/status",
		"method":   "GET",
		"clientIP": c.RealIP(),
	})

	requestLogger.Debug("Health check requested")

	workingDir, _ := os.Getwd()

	// Include memory store statistics
	memoryStats := s.memoryStore.GetSessionStats()

	// Include active executions
	activeExecutions := s.cancelManager.GetActiveExecutions()

	response := map[string]interface{}{
		"status":           "healthy",
		"workingDir":       workingDir,
		"memory":           memoryStats,
		"activeExecutions": activeExecutions,
		"executionCount":   len(activeExecutions),
	}

	requestLogger.WithFields(logrus.Fields{
		"activeExecutions": len(activeExecutions),
		"sessions":         memoryStats["totalSessions"],
	}).Debug("Status check completed")

	return c.JSON(http.StatusOK, response)
}

// handleGetSession returns information about a specific chat session
func (s *Server) handleGetSession(c echo.Context) error {
	sessionID := c.Param("sessionId")

	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint":  "/sessions/:sessionId",
		"method":    "GET",
		"sessionID": sessionID,
		"clientIP":  c.RealIP(),
	})

	if sessionID == "" {
		requestLogger.Warn("Session ID not provided")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session ID required"})
	}

	// Try to get the session (don't create if it doesn't exist)
	session, exists := s.memoryStore.GetSession(sessionID)

	if !exists {
		requestLogger.Warn("Session not found")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	session.mutex.RLock()
	sessionInfo := map[string]interface{}{
		"id":           session.ID,
		"created":      session.Created,
		"updated":      session.Updated,
		"messageCount": len(session.Messages),
		"messages":     session.Messages,
	}
	session.mutex.RUnlock()

	requestLogger.WithField("messageCount", len(session.Messages)).Info("Session information retrieved")

	return c.JSON(http.StatusOK, sessionInfo)
}

// handleClearSession clears the history of a specific chat session
func (s *Server) handleClearSession(c echo.Context) error {
	sessionID := c.Param("sessionId")

	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint":  "/sessions/:sessionId/clear",
		"method":    "POST",
		"sessionID": sessionID,
		"clientIP":  c.RealIP(),
	})

	if sessionID == "" {
		requestLogger.Warn("Session ID not provided for clearing")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session ID required"})
	}

	// Try to get the session
	session, exists := s.memoryStore.GetSession(sessionID)

	if !exists {
		requestLogger.Warn("Session not found for clearing")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	// Clear the session messages
	messageCount := session.ClearMessages()

	requestLogger.WithField("clearedMessages", messageCount).Info("Session cleared successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":         "Session cleared successfully",
		"sessionId":       sessionID,
		"clearedMessages": messageCount,
	})
}

// handleDeleteSession deletes a specific chat session
func (s *Server) handleDeleteSession(c echo.Context) error {
	sessionID := c.Param("sessionId")

	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint":  "/sessions/:sessionId",
		"method":    "DELETE",
		"sessionID": sessionID,
		"clientIP":  c.RealIP(),
	})

	if sessionID == "" {
		requestLogger.Warn("Session ID not provided for deletion")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session ID required"})
	}

	// Try to delete the session
	exists := s.memoryStore.DeleteSession(sessionID)

	if !exists {
		requestLogger.Warn("Session not found for deletion")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	requestLogger.Info("Session deleted successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Session deleted successfully",
		"sessionId": sessionID,
	})
}

// handleListSessions returns a list of all active sessions
func (s *Server) handleListSessions(c echo.Context) error {
	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint": "/sessions",
		"method":   "GET",
		"clientIP": c.RealIP(),
	})

	requestLogger.Debug("Listing all sessions")

	sessions := s.memoryStore.GetAllSessions()

	requestLogger.WithField("sessionCount", len(sessions)).Info("Sessions listed successfully")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
}

func (s *Server) handleStopExecution(c echo.Context) error {
	requestLogger := s.logger.WithFields(logrus.Fields{
		"endpoint": "/stop",
		"method":   "POST",
		"clientIP": c.RealIP(),
	})

	requestLogger.Info("Received stop execution request")

	var req StopRequest
	if err := c.Bind(&req); err != nil {
		requestLogger.WithError(err).Error("Failed to parse stop request body")
		return c.JSON(http.StatusBadRequest, StopResponse{
			Success: false,
			Message: "Invalid request format",
			Stopped: false,
		})
	}

	if req.ExecutionID == "" {
		requestLogger.Error("Empty execution ID in stop request")
		return c.JSON(http.StatusBadRequest, StopResponse{
			Success: false,
			Message: "Execution ID is required",
			Stopped: false,
		})
	}

	requestLogger.WithField("executionID", req.ExecutionID).Info("Attempting to stop execution")

	// Try to cancel the execution
	stopped := s.cancelManager.CancelExecution(req.ExecutionID)

	if stopped {
		requestLogger.WithField("executionID", req.ExecutionID).Info("Execution stopped successfully")
		return c.JSON(http.StatusOK, StopResponse{
			Success: true,
			Message: "Execution stopped successfully",
			Stopped: true,
		})
	} else {
		requestLogger.WithField("executionID", req.ExecutionID).Warn("Execution not found or already completed")
		return c.JSON(http.StatusNotFound, StopResponse{
			Success: false,
			Message: "Execution not found or already completed",
			Stopped: false,
		})
	}
}

// RegisterRoutes registers all HTTP routes for the server
func (s *Server) RegisterRoutes(e *echo.Echo) {
	s.logger.Info("Registering routes")

	// API routes
	e.POST("/chat", s.handleChat)
	e.POST("/chat/stream", s.handleStreamChat)
	e.GET("/status", s.handleStatus)

	// Session management routes
	e.GET("/sessions", s.handleListSessions)
	e.GET("/sessions/:sessionId", s.handleGetSession)
	e.POST("/sessions/:sessionId/clear", s.handleClearSession)
	e.DELETE("/sessions/:sessionId", s.handleDeleteSession)
	e.POST("/stop", s.handleStopExecution)

	// Serve static files
	e.Static("/", "static")
	s.logger.Info("Routes registered successfully")
}
