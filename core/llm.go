/*
Package core provides LLM integration and response processing for the Skynet Agent application.

This file implements a wrapper around language model implementations that provides:
- Response cleaning and sanitization to remove unwanted formatting tags
- Agent format validation and correction
- Logging and monitoring of LLM interactions
- Error handling and fallback mechanisms for robust operation

The CleaningLLMWrapper ensures that LLM responses are properly formatted for
agent execution while maintaining compatibility with the langchaingo interface.
*/
package core

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
)

// CleaningLLMWrapper is a custom LLM wrapper that preprocesses and cleans responses
// from language models to ensure proper formatting for agent execution.
// It acts as a middleware layer between the agent framework and the underlying LLM,
// providing response sanitization and format correction.
type CleaningLLMWrapper struct {
	wrappedLLM llms.Model     // The underlying LLM implementation to wrap
	config     *Config        // Application configuration for behavior control
	logger     *logrus.Logger // Structured logger for monitoring and debugging
}

// NewCleaningLLMWrapper creates a new instance of the cleaning LLM wrapper.
// This factory function initializes the wrapper with the necessary dependencies
// for response processing and logging.
//
// Parameters:
//   - llm: The underlying language model to wrap
//   - config: Application configuration containing processing parameters
//   - logger: Logger instance for monitoring LLM interactions
//
// Returns:
//   - *CleaningLLMWrapper: Configured wrapper ready for use
func NewCleaningLLMWrapper(llm llms.Model, config *Config, logger *logrus.Logger) *CleaningLLMWrapper {
	return &CleaningLLMWrapper{
		wrappedLLM: llm,
		config:     config,
		logger:     logger,
	}
}

// truncateForLog truncates text to a configurable length for logging purposes.
// This prevents excessive log output while maintaining useful information for debugging.
// The truncation length is controlled by the configuration to balance detail and readability.
//
// Parameters:
//   - text: The text content to potentially truncate
//
// Returns:
//   - string: Truncated text with ellipsis if truncation occurred
func (w *CleaningLLMWrapper) truncateForLog(text string) string {
	if len(text) <= w.config.LogTruncateLength {
		return text
	}
	return text[:w.config.LogTruncateLength] + "..."
}

// cleanAgentResponse processes and cleans LLM responses to ensure proper agent execution format.
// This method handles various common issues in LLM responses including:
// - Removing thinking/reasoning tags that interfere with parsing
// - Cleaning up excessive whitespace and formatting
// - Detecting and correcting non-agent formatted responses
// - Providing fallback responses for empty or problematic content
//
// The cleaning process ensures that responses are compatible with agent execution
// frameworks while preserving the actual content and intent.
//
// Parameters:
//   - response: Raw response from the LLM that needs cleaning
//
// Returns:
//   - string: Cleaned and formatted response ready for agent execution
func (w *CleaningLLMWrapper) cleanAgentResponse(response string) string {
	// Remove <think> tags and their content more robustly
	// This regex matches the opening <think> tag, any content (including newlines), and the closing </think> tag
	thinkRegex := regexp.MustCompile(`(?i)(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")

	// Also remove any standalone think tags that might not be properly closed
	openThinkRegex := regexp.MustCompile(`(?i)<think>.*`)
	cleaned = openThinkRegex.ReplaceAllString(cleaned, "")

	// Remove any other common problematic tags that might interfere with parsing
	reasoningRegex := regexp.MustCompile(`(?i)(?s)<reasoning>.*?</reasoning>`)
	cleaned = reasoningRegex.ReplaceAllString(cleaned, "")

	// Clean up extra whitespace and newlines that might be left after tag removal
	cleaned = strings.TrimSpace(cleaned)

	// Remove multiple consecutive newlines to improve readability
	multiNewlineRegex := regexp.MustCompile(`\n\s*\n\s*\n+`)
	cleaned = multiNewlineRegex.ReplaceAllString(cleaned, "\n\n")

	// Fix empty Action Input fields that cause parsing errors
	// The langchaingo framework requires Action Input to have a value
	emptyActionInputRegex := regexp.MustCompile(`(?m)^Action Input:\s*$`)
	if emptyActionInputRegex.MatchString(cleaned) {
		w.logger.Debug("Detected empty Action Input field, adding empty string value")
		cleaned = emptyActionInputRegex.ReplaceAllString(cleaned, "Action Input: ")
	}

	// Also handle cases where Action Input is followed by newline/whitespace only
	actionInputEndRegex := regexp.MustCompile(`(?m)^Action Input:\s*\n`)
	if actionInputEndRegex.MatchString(cleaned) {
		w.logger.Debug("Detected Action Input followed by newline only, adding empty string value")
		cleaned = actionInputEndRegex.ReplaceAllString(cleaned, "Action Input: \n")
	}

	// Check if this looks like a direct response (doesn't follow agent format)
	// Agent format should contain specific keywords like "Thought:", "Action:", "Final Answer:" etc.
	hasAgentFormat := strings.Contains(cleaned, "Thought:") ||
		strings.Contains(cleaned, "Action:") ||
		strings.Contains(cleaned, "Final Answer:") ||
		strings.Contains(cleaned, "Observation:")

	// If it doesn't follow agent format and looks like a direct answer, wrap it appropriately
	if !hasAgentFormat && cleaned != "" {
		// Check if it looks like a substantial response (not just an error or short text)
		if len(cleaned) > 50 && !strings.Contains(strings.ToLower(cleaned), "i don't") {
			w.logger.WithFields(logrus.Fields{
				"originalLength": len(response),
				"cleanedLength":  len(cleaned),
				"wrapped":        true,
			}).Info("Wrapping direct response in Final Answer format")

			// Wrap the direct response in proper agent format for consistent processing
			cleaned = fmt.Sprintf("Thought: I can provide a direct answer to this question.\nFinal Answer: %s", cleaned)
		}
	}

	// If the response is empty after cleaning, return a helpful fallback message
	if cleaned == "" {
		return "I understand your request but need to process it differently. Could you please rephrase your question?"
	}

	return cleaned
}

// GenerateContent implements the langchaingo LLM interface for content generation.
// This method wraps the underlying LLM's GenerateContent call and applies response
// cleaning to all generated choices. It maintains full compatibility with the
// langchaingo interface while providing enhanced response processing.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - messages: Input messages for content generation
//   - options: Additional call options for LLM configuration
//
// Returns:
//   - *llms.ContentResponse: Cleaned response with processed content choices
//   - error: Any error from the underlying LLM or processing
func (w *CleaningLLMWrapper) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// Call the underlying LLM for content generation
	response, err := w.wrappedLLM.GenerateContent(ctx, messages, options...)
	if err != nil {
		return response, err
	}

	// Clean the response content for each choice
	if response != nil && len(response.Choices) > 0 {
		for i := range response.Choices {
			original := response.Choices[i].Content
			cleaned := w.cleanAgentResponse(original)
			response.Choices[i].Content = cleaned

			// Log if significant cleaning occurred for monitoring purposes
			if len(original) != len(cleaned) {
				w.logger.WithFields(logrus.Fields{
					"originalLength":  len(original),
					"cleanedLength":   len(cleaned),
					"originalPreview": w.truncateForLog(original),
				}).Debug("Cleaned LLM response content")
			}
		}
	}

	return response, nil
}

// Call implements the langchaingo LLM interface for simple string-based calls.
// This method wraps the underlying LLM's Call method and applies response cleaning
// to ensure consistent output formatting. It's typically used for simpler interactions
// that don't require complex message structures.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - prompt: Input prompt string for generation
//   - options: Additional call options for LLM configuration
//
// Returns:
//   - string: Cleaned response string ready for use
//   - error: Any error from the underlying LLM or processing
func (w *CleaningLLMWrapper) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	// Call the underlying LLM with the provided prompt
	response, err := w.wrappedLLM.Call(ctx, prompt, options...)
	if err != nil {
		return response, err
	}

	// Clean the response using the same processing logic
	cleaned := w.cleanAgentResponse(response)

	// Log if significant cleaning occurred for debugging and monitoring
	if len(response) != len(cleaned) {
		w.logger.WithFields(logrus.Fields{
			"originalLength":  len(response),
			"cleanedLength":   len(cleaned),
			"originalPreview": w.truncateForLog(response),
		}).Debug("Cleaned LLM call response")
	}

	return cleaned, nil
}

// CleanAgentResponse provides external access to the response cleaning functionality.
// This method allows other components to benefit from the same response processing
// logic without needing to wrap LLM calls directly. It's useful for post-processing
// responses that have been obtained through other means.
//
// Parameters:
//   - response: Raw response string to be cleaned
//
// Returns:
//   - string: Cleaned and formatted response
func (w *CleaningLLMWrapper) CleanAgentResponse(response string) string {
	return w.cleanAgentResponse(response)
}
