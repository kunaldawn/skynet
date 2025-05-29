package core

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// Custom callback handler for verbose logging
type VerboseCallbackHandler struct {
	requestLogger *logrus.Entry
	iteration     int
	step          int
	config        *Config
}

func NewVerboseCallbackHandler(requestLogger *logrus.Entry, config *Config) *VerboseCallbackHandler {
	return &VerboseCallbackHandler{
		requestLogger: requestLogger,
		iteration:     0,
		step:          0,
		config:        config,
	}
}

// StreamingCallbackHandler extends VerboseCallbackHandler to stream debug info to client
type StreamingCallbackHandler struct {
	*VerboseCallbackHandler
	streamFunc func(msg StreamMessage)
}

func NewStreamingCallbackHandler(requestLogger *logrus.Entry, config *Config, streamFunc func(msg StreamMessage)) *StreamingCallbackHandler {
	return &StreamingCallbackHandler{
		VerboseCallbackHandler: NewVerboseCallbackHandler(requestLogger, config),
		streamFunc:             streamFunc,
	}
}

// Helper function to truncate text for logging with configurable length
func (h *VerboseCallbackHandler) truncateForLog(text string) string {
	if len(text) <= h.config.LogTruncateLength {
		return text
	}
	return text[:h.config.LogTruncateLength] + "..."
}

func (h *VerboseCallbackHandler) HandleText(ctx context.Context, text string) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":  h.iteration,
		"step":       h.step,
		"text":       h.truncateForLog(text),
		"textLength": len(text),
	}).Debug("Agent processing text")
}

func (h *VerboseCallbackHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	h.iteration++
	h.step = 0 // Reset step counter for new iteration
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":   h.iteration,
		"step":        h.step,
		"promptCount": len(prompts),
		"firstPrompt": func() string {
			if len(prompts) > 0 {
				return h.truncateForLog(prompts[0])
			}
			return ""
		}(),
	}).Info("Agent iteration started - LLM call beginning")
}

func (h *VerboseCallbackHandler) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":    h.iteration,
		"step":         h.step,
		"messageCount": len(ms),
	}).Info("LLM content generation started")
}

func (h *VerboseCallbackHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"response": func() string {
			if res != nil && len(res.Choices) > 0 && res.Choices[0].Content != "" {
				return h.truncateForLog(res.Choices[0].Content)
			}
			return ""
		}(),
	}).Info("LLM content generation completed")
}

func (h *VerboseCallbackHandler) HandleLLMError(ctx context.Context, err error) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"error":     err.Error(),
	}).Error("LLM call failed")
}

func (h *VerboseCallbackHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"inputs":    inputs,
	}).Info("Agent chain execution started")
}

func (h *VerboseCallbackHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":       h.iteration,
		"step":            h.step,
		"outputs":         outputs,
		"totalIterations": h.iteration,
	}).Info("Agent chain execution completed")
}

func (h *VerboseCallbackHandler) HandleChainError(ctx context.Context, err error) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":       h.iteration,
		"step":            h.step,
		"error":           err.Error(),
		"totalIterations": h.iteration,
	}).Error("Agent chain execution failed")
}

func (h *VerboseCallbackHandler) HandleToolStart(ctx context.Context, input string) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"input":     input,
	}).Info("Tool execution started")
}

func (h *VerboseCallbackHandler) HandleToolEnd(ctx context.Context, output string) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":    h.iteration,
		"step":         h.step,
		"output":       h.truncateForLog(output),
		"outputLength": len(output),
	}).Info("Tool execution completed")
}

func (h *VerboseCallbackHandler) HandleToolError(ctx context.Context, err error) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"error":     err.Error(),
	}).Error("Tool execution failed")
}

func (h *VerboseCallbackHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"action":    action.Tool,
		"input":     action.ToolInput,
		"reasoning": action.Log,
	}).Info("Agent decided on action")
}

func (h *VerboseCallbackHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"finalResponse": func() string {
			if output, ok := finish.ReturnValues["output"].(string); ok {
				return h.truncateForLog(output)
			}
			return ""
		}(),
		"reasoning":       finish.Log,
		"totalIterations": h.iteration,
	}).Info("Agent finished successfully")
}

func (h *VerboseCallbackHandler) HandleRetrieverStart(ctx context.Context, query string) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"query":     query,
	}).Debug("Retriever started")
}

func (h *VerboseCallbackHandler) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration":     h.iteration,
		"step":          h.step,
		"query":         query,
		"documentCount": len(documents),
	}).Debug("Retriever completed")
}

func (h *VerboseCallbackHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	h.requestLogger.WithFields(logrus.Fields{
		"iteration": h.iteration,
		"step":      h.step,
		"chunkSize": len(chunk),
	}).Debug("Streaming chunk received")
}

// Streaming callback handler implementations
func (h *StreamingCallbackHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	h.VerboseCallbackHandler.HandleLLMStart(ctx, prompts)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "LLM call started",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("llm_start_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber":  h.step,
				"promptCount": len(prompts),
				"firstPromptPreview": func() string {
					if len(prompts) > 0 {
						return h.truncateForLog(prompts[0])
					}
					return ""
				}(),
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	h.VerboseCallbackHandler.HandleLLMGenerateContentEnd(ctx, res)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		content := ""
		if res != nil && len(res.Choices) > 0 {
			content = res.Choices[0].Content
		}

		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "LLM response generated",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("llm_response_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber":      h.step,
				"responseLength":  len(content),
				"responsePreview": h.truncateForLog(content),
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	h.VerboseCallbackHandler.HandleChainStart(ctx, inputs)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "Agent chain execution started",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("chain_start_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber": h.step,
				"inputs":     inputs,
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	h.VerboseCallbackHandler.HandleChainEnd(ctx, outputs)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "Agent chain execution completed",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("chain_end_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber":      h.step,
				"outputs":         outputs,
				"totalIterations": h.iteration,
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleToolStart(ctx context.Context, input string) {
	h.VerboseCallbackHandler.HandleToolStart(ctx, input)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "Tool execution started",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("tool_start_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber": h.step,
				"toolInput":  input,
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleToolEnd(ctx context.Context, output string) {
	h.VerboseCallbackHandler.HandleToolEnd(ctx, output)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "Tool execution completed",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("tool_end_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber":   h.step,
				"toolOutput":   h.truncateForLog(output),
				"outputLength": len(output),
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	h.VerboseCallbackHandler.HandleAgentAction(ctx, action)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   fmt.Sprintf("Agent chose to use tool: %s", action.Tool),
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("agent_action_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber": h.step,
				"tool":       action.Tool,
				"toolInput":  action.ToolInput,
				"reasoning":  action.Log,
			},
		})
	}
}

func (h *StreamingCallbackHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	h.VerboseCallbackHandler.HandleAgentFinish(ctx, finish)
	h.step++ // Increment step for each action

	// Send debug info to client
	if h.streamFunc != nil {
		finalResponse := ""
		if output, ok := finish.ReturnValues["output"].(string); ok {
			finalResponse = output
		}

		h.streamFunc(StreamMessage{
			Type:      "debug",
			Content:   "Agent finished successfully",
			Debug:     true,
			Iteration: h.iteration,
			Step:      fmt.Sprintf("agent_finish_%d", h.step),
			Details: map[string]interface{}{
				"stepNumber":      h.step,
				"finalResponse":   finalResponse,
				"reasoning":       finish.Log,
				"totalIterations": h.iteration,
			},
		})
	}
}
