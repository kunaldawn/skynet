/**
 * Skynet Agent Chat Application
 * 
 * This file implements the frontend web interface for the Skynet Agent application.
 * It provides a modern, responsive chat interface with the following key features:
 * 
 * Core Features:
 * - Real-time WebSocket communication with the agent
 * - Plain text display of agent responses
 * - Live streaming of agent responses and thinking processes
 * - Debug mode for viewing internal agent operations
 * - Responsive design with automatic text area resizing
 * - Status monitoring and connection health checks
 * - Execution control (start/stop operations)
 * 
 * Architecture:
 * - Single-page application using vanilla JavaScript
 * - WebSocket-based real-time communication
 * - Progressive enhancement with fallback support
 * - Plain text rendering with line break preservation
 * 
 * The application maintains state for the current conversation, handles streaming
 * responses from the agent, and provides a ChatGPT-like interface for users.
 */

/**
 * ChatApp - Main application class that manages the chat interface
 * 
 * This class handles all aspects of the chat application including:
 * - UI state management and event handling
 * - WebSocket communication with the backend
 * - Message rendering with plain text display
 * - Real-time streaming of agent responses
 * - Debug information display and execution control
 */
class ChatApp {
    /**
     * Initialize the ChatApp with DOM element references and configuration
     * 
     * Sets up the application state, DOM references, and initializes all
     * necessary components including event listeners.
     */
    constructor() {
        // Core DOM element references for the chat interface
        this.messagesContainer = document.getElementById('chat-messages');
        this.chatForm = document.getElementById('chat-form');
        this.chatInput = document.getElementById('chat-input');
        this.sendButton = document.getElementById('send-button');
        this.stopButton = document.getElementById('stop-button');
        this.statusText = document.getElementById('status-text');

        // Application state variables
        this.isTyping = false;                    // Prevents multiple simultaneous requests
        this.debugMode = true;                    // Always enabled for development visibility
        this.currentExecutionId = null;          // Track current agent execution for stop functionality
        this.currentResponseMessage = null;      // Reference to the currently streaming message element
        this.accumulatedContent = '';            // Accumulate streaming content for proper rendering
        
        // Initialize the application components
        this.init();
    }

    /**
     * Initialize the application by setting up event listeners and initial state
     * 
     * Configures all necessary event handlers for form submission, keyboard shortcuts,
     * auto-resizing, and button interactions. Also performs initial setup tasks.
     */
    init() {
        // Set up core event listeners
        this.chatForm.addEventListener('submit', (e) => this.handleSubmit(e));
        this.chatInput.addEventListener('keydown', (e) => this.handleKeyDown(e));
        this.chatInput.addEventListener('input', () => this.autoResize());
        this.stopButton.addEventListener('click', () => this.handleStop());
        
        // Perform initial setup tasks
        this.checkStatus();                    // Check backend connectivity
        this.addClickHandlersToSuggestions(); // Enable suggestion interaction
    }

    /**
     * Add click handlers to suggestion elements for quick message input
     * 
     * Enables users to click on suggestion items to quickly populate the input
     * field with common queries or commands.
     */
    addClickHandlersToSuggestions() {
        const suggestions = document.querySelectorAll('.suggestion-item');
        suggestions.forEach(suggestion => {
            suggestion.addEventListener('click', () => {
                // Extract text content without emoji prefix
                const text = suggestion.textContent.replace(/^[^\s]+ /, '');
                this.chatInput.value = text;
                this.chatInput.focus();
            });
        });
    }

    /**
     * Handle keyboard shortcuts in the input field
     * 
     * Implements common chat interface shortcuts:
     * - Enter: Send message (without Shift)
     * - Shift+Enter: Add line break
     * 
     * @param {KeyboardEvent} e - The keyboard event to handle
     */
    handleKeyDown(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            this.chatForm.dispatchEvent(new Event('submit'));
        }
    }

    /**
     * Auto-resize the input textarea based on content
     * 
     * Dynamically adjusts the height of the input field to accommodate
     * multi-line input while maintaining a maximum height for usability.
     */
    autoResize() {
        this.chatInput.style.height = 'auto';
        this.chatInput.style.height = Math.min(this.chatInput.scrollHeight, 120) + 'px';
    }

    /**
     * Handle form submission and message sending
     * 
     * Processes user input, validates it, adds it to the chat, and initiates
     * the agent communication process. Prevents submission during active conversations.
     * 
     * @param {Event} e - The form submission event
     */
    async handleSubmit(e) {
        e.preventDefault();
        
        const message = this.chatInput.value.trim();
        // Prevent empty messages or submissions during active conversations
        if (!message || this.isTyping) return;

        // Add user message to the chat interface
        this.addMessage(message, 'user');
        this.chatInput.value = '';
        this.autoResize();
        this.setTyping(true);

        try {
            await this.sendMessage(message);
        } catch (error) {
            this.addMessage(`Error: ${error.message}`, 'assistant error');
        } finally {
            this.setTyping(false);
        }
    }

    addMessage(content, className) {
        // Remove welcome message when first user message is added
        if (className === 'user') {
            this.removeWelcomeMessage();
        }
        
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${className}`;
        
        // Display plain text with line breaks preserved
        const textWithBreaks = content.replace(/\n/g, '<br>');
        messageDiv.innerHTML = textWithBreaks;

        this.messagesContainer.appendChild(messageDiv);
        this.scrollToBottom();
    }

    removeWelcomeMessage() {
        const welcomeMessage = document.querySelector('.message.assistant.welcome-message');
        if (welcomeMessage) {
            welcomeMessage.remove();
        }
    }

    setTyping(typing) {
        this.isTyping = typing;
        this.sendButton.disabled = typing;
        this.chatInput.disabled = typing;

        // Show/hide stop button based on typing state
        if (typing) {
            this.stopButton.style.display = 'flex';
            this.sendButton.style.display = 'none';
        } else {
            this.stopButton.style.display = 'none';
            this.sendButton.style.display = 'flex';
        }

        if (typing) {
            const typingDiv = document.createElement('div');
            typingDiv.className = 'message typing';
            typingDiv.id = 'typing-indicator';
            typingDiv.innerHTML = `
                <span>AI is analyzing your request</span>
                <div class="typing-indicator">
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                </div>
            `;
            this.messagesContainer.appendChild(typingDiv);
            this.scrollToBottom();
        } else {
            const typingIndicator = document.getElementById('typing-indicator');
            if (typingIndicator) {
                typingIndicator.remove();
            }
        }
    }

    async handleStop() {
        if (!this.currentExecutionId) {
            console.log('No execution to stop');
            return;
        }

        try {
            const response = await fetch('/stop', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ 
                    executionId: this.currentExecutionId 
                })
            });

            const result = await response.json();
            
            if (result.success) {
                console.log('Execution stopped successfully');
                this.addMessage('Execution stopped by user', 'assistant');
            } else {
                console.log('Failed to stop execution:', result.message);
                this.addMessage(`Stop failed: ${result.message}`, 'assistant error');
            }
        } catch (error) {
            console.error('Error stopping execution:', error);
            this.addMessage(`Error stopping execution: ${error.message}`, 'assistant error');
        } finally {
            this.currentExecutionId = null;
            this.currentResponseMessage = null;
            this.accumulatedContent = '';
            this.setTyping(false);
        }
    }

    async sendMessage(message) {
        try {
            // Reset streaming state
            this.currentResponseMessage = null;
            this.accumulatedContent = '';
            
            const response = await fetch('/chat/stream', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ 
                    message: message,
                    debug: this.debugMode // Always true now
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                const { done, value } = await reader.read();
                
                if (done) break;
                
                buffer += decoder.decode(value, { stream: true });
                
                // Process complete SSE messages
                const lines = buffer.split('\n');
                buffer = lines.pop(); // Keep incomplete line in buffer
                
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = line.slice(6); // Remove 'data: ' prefix
                        if (data.trim()) {
                            try {
                                const parsed = JSON.parse(data);
                                this.handleStreamMessage(parsed);
                            } catch (error) {
                                console.error('Error parsing stream data:', error);
                            }
                        }
                    }
                }
            }
            
            this.setTyping(false);
        } catch (error) {
            console.error('Error in stream:', error);
            // Clean up streaming state on error
            this.currentResponseMessage = null;
            this.accumulatedContent = '';
            this.addMessage(`Error: ${error.message}`, 'assistant error');
            this.setTyping(false);
        }
    }

    handleStreamMessage(data) {
        switch (data.type) {
            case 'execution_started':
                this.currentExecutionId = data.content;
                console.log('Execution started with ID:', this.currentExecutionId);
                break;
                
            case 'thinking':
                this.updateTypingMessage(data.content);
                break;
                
            case 'debug':
                // Always show debug messages now
                this.addDebugMessage(data);
                break;
                
            case 'response':
                // Handle streaming response with plain text rendering
                if (!this.currentResponseMessage) {
                    // Create new response message container
                    this.removeWelcomeMessage();
                    this.currentResponseMessage = document.createElement('div');
                    this.currentResponseMessage.className = 'message assistant';
                    this.messagesContainer.appendChild(this.currentResponseMessage);
                    this.accumulatedContent = '';
                }
                
                // Accumulate content
                this.accumulatedContent += data.content;
                
                // Render accumulated plain text with line breaks
                this.renderPlainTextToElement(this.currentResponseMessage, this.accumulatedContent);
                
                if (data.complete) {
                    // Finalize the message
                    this.currentResponseMessage = null;
                    this.accumulatedContent = '';
                    this.currentExecutionId = null;
                    this.setTyping(false);
                }
                
                this.scrollToBottom();
                break;
                
            case 'stopped':
                this.addMessage(data.content, 'assistant');
                this.currentResponseMessage = null;
                this.accumulatedContent = '';
                this.currentExecutionId = null;
                this.setTyping(false);
                break;
                
            case 'error':
                this.addMessage(`Error: ${data.content}`, 'assistant error');
                this.currentExecutionId = null;
                this.setTyping(false);
                break;
        }
        
        this.scrollToBottom();
    }

    updateTypingMessage(content) {
        const typingIndicator = document.getElementById('typing-indicator');
        if (typingIndicator) {
            typingIndicator.innerHTML = `
                <span>${content}</span>
                <div class="typing-indicator">
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                    <div class="typing-dot"></div>
                </div>
            `;
        }
    }

    addDebugMessage(data) {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message debug';
        
        // Enhanced header with step and iteration - show both iteration and step number
        const headerP = document.createElement('p');
        headerP.className = 'debug-header';
        
        // Use step name from data.step, and show iteration properly
        const stepName = data.step || 'debug';
        const stepNum = data.details && data.details.stepNumber !== undefined ? data.details.stepNumber : 0;
        
        headerP.innerHTML = `Step ${stepNum}`;
        
        // Enhanced content paragraph - handle line breaks properly
        const contentP = document.createElement('p');
        contentP.className = 'debug-content';
        // Convert \n to actual line breaks (handle both \\n and \n) but avoid excessive spacing
        const contentWithBreaks = data.content.replace(/\\n/g, '\n').replace(/\n+/g, '<br>');
        contentP.innerHTML = contentWithBreaks;
        
        messageDiv.appendChild(headerP);
        messageDiv.appendChild(contentP);
        
        // Enhanced details if available
        if (data.details && Object.keys(data.details).length > 0) {
            const detailsDiv = document.createElement('div');
            detailsDiv.className = 'debug-details';
            
            for (const [key, value] of Object.entries(data.details)) {
                // Skip stepNumber in details since we show it in header
                if (key === 'stepNumber') continue;
                
                const detailP = document.createElement('p');
                detailP.innerHTML = `<span class="debug-key">${key}:</span> <span class="debug-value">${this.formatSimpleValue(value)}</span>`;
                detailsDiv.appendChild(detailP);
            }
            
            messageDiv.appendChild(detailsDiv);
        }
        
        this.messagesContainer.appendChild(messageDiv);
        this.scrollToBottom();
    }

    formatSimpleValue(value) {
        // Don't trim debug messages - show full content
        if (typeof value === 'object') {
            // For JSON objects, preserve formatting but handle line breaks more carefully
            const jsonStr = JSON.stringify(value, null, 2);
            // Only convert literal \n to actual newlines, then to breaks more conservatively
            return jsonStr.replace(/\\n/g, '\n').replace(/\n+/g, '<br>').replace(/ /g, '&nbsp;');
        }
        // For strings, handle line breaks more conservatively
        return String(value).replace(/\\n/g, '\n').replace(/\n+/g, '<br>');
    }

    async checkStatus() {
        try {
            const response = await fetch('/status');
            if (response.ok) {
                this.statusText.textContent = 'Online';
            } else {
                this.statusText.textContent = 'Offline';
            }
        } catch (error) {
            this.statusText.textContent = 'Offline';
        }
    }

    scrollToBottom() {
        setTimeout(() => {
            this.messagesContainer.scrollTop = this.messagesContainer.scrollHeight;
        }, 50);
    }

    /**
     * Render plain text content to a specific element with line breaks preserved
     * 
     * @param {HTMLElement} element - The element to render text content into
     * @param {string} content - The plain text content to render
     */
    renderPlainTextToElement(element, content) {
        // Convert newlines to HTML line breaks
        const textWithBreaks = content.replace(/\n/g, '<br>');
        element.innerHTML = textWithBreaks;
    }
}

// Initialize the chat app when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new ChatApp();
}); 