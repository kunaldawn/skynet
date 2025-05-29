# Environment Configuration

This document describes all the environment variables you can use to configure the Skynet Agent system.

## Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port number for the HTTP server |

## LLM Provider Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `ollama` | LLM provider to use: `ollama` or `gemini` |

## Ollama Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_ENDPOINT` | `http://localhost:11434` | URL endpoint for the Ollama server |
| `OLLAMA_MODEL` | `qwen3` | Model name to use with Ollama |

## Google Gemini Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_API_KEY` | (required) | Google Gemini API key (required when `LLM_PROVIDER=gemini`) |
| `GEMINI_MODEL` | `gemini-1.5-pro` | Gemini model name: `gemini-1.5-pro`, `gemini-1.5-flash`, `gemini-pro` |

> **Note**: To use Gemini, get your API key from [Google AI Studio](https://ai.google.dev/)

## Agent Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_ITERATIONS` | `100` | Maximum number of iterations the agent can perform per request |
| `REQUEST_TIMEOUT` | `300` | Request timeout in seconds (5 minutes default) |
| `CONTEXT_LIMIT` | `10` | Maximum number of previous messages to include in conversation context |

## Memory Store Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_MAX_AGE_HOURS` | `24` | Maximum age of chat sessions in hours before they expire |
| `CLEANUP_INTERVAL_MINUTES` | `60` | How often to clean up expired sessions (in minutes) |
| `MAX_SESSIONS_PER_USER` | `50` | Maximum number of sessions per user (future use) |

## Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `LOG_TRUNCATE_LENGTH` | `500` | Maximum length for log message truncation |
| `DEBUG_MODE` | `false` | Enable debug mode for enhanced logging (`true` or `false`) |

## Performance Tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_CONCURRENT_REQUESTS` | `100` | Maximum number of concurrent requests (future use) |

## Example Configuration

Create a `.env` file or set environment variables:

### Using Ollama (Local Deployment)

```bash
# Server
export PORT=8080

# LLM Provider
export LLM_PROVIDER=ollama

# Ollama
export OLLAMA_ENDPOINT=http://localhost:11434
export OLLAMA_MODEL=qwen3

# Agent behavior
export MAX_ITERATIONS=50
export REQUEST_TIMEOUT=600
export CONTEXT_LIMIT=5

# Memory management
export SESSION_MAX_AGE_HOURS=12
export CLEANUP_INTERVAL_MINUTES=30

# Logging
export LOG_LEVEL=debug
export LOG_TRUNCATE_LENGTH=300
export DEBUG_MODE=true
```

### Using Google Gemini (Cloud Deployment)

```bash
# Server
export PORT=8080

# LLM Provider
export LLM_PROVIDER=gemini

# Gemini
export GEMINI_API_KEY=AIzaSyC1234567890abcdefghijklmnopqrstuvwxyz
export GEMINI_MODEL=gemini-1.5-pro

# Agent behavior
export MAX_ITERATIONS=50
export REQUEST_TIMEOUT=600
export CONTEXT_LIMIT=5

# Memory management
export SESSION_MAX_AGE_HOURS=12
export CLEANUP_INTERVAL_MINUTES=30

# Logging
export LOG_LEVEL=debug
export LOG_TRUNCATE_LENGTH=300
export DEBUG_MODE=true
```

## Docker Configuration

When running in Docker, you can pass environment variables using the `-e` flag:

### Using Ollama with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e LLM_PROVIDER=ollama \
  -e OLLAMA_ENDPOINT=http://ollama:11434 \
  -e OLLAMA_MODEL=qwen3 \
  -e MAX_ITERATIONS=50 \
  -e REQUEST_TIMEOUT=600 \
  -e LOG_LEVEL=debug \
  skynet-agent
```

### Using Gemini with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e LLM_PROVIDER=gemini \
  -e GEMINI_API_KEY=AIzaSyC1234567890abcdefghijklmnopqrstuvwxyz \
  -e GEMINI_MODEL=gemini-1.5-pro \
  -e MAX_ITERATIONS=50 \
  -e REQUEST_TIMEOUT=600 \
  -e LOG_LEVEL=debug \
  skynet-agent
```

## Docker Compose Configuration

### Example with Ollama Service

```yaml
version: '3.8'
services:
  skynet-agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LLM_PROVIDER=ollama
      - OLLAMA_ENDPOINT=http://ollama:11434
      - OLLAMA_MODEL=qwen3
      - MAX_ITERATIONS=75
      - REQUEST_TIMEOUT=900
      - CONTEXT_LIMIT=8
      - SESSION_MAX_AGE_HOURS=48
      - CLEANUP_INTERVAL_MINUTES=120
      - LOG_LEVEL=info
      - LOG_TRUNCATE_LENGTH=400
      - DEBUG_MODE=false
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama
    ports:
      - "11434:11434"
```

### Example with Gemini (Cloud)

```yaml
version: '3.8'
services:
  skynet-agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LLM_PROVIDER=gemini
      - GEMINI_API_KEY=${GEMINI_API_KEY}  # Set in .env file
      - GEMINI_MODEL=gemini-1.5-pro
      - MAX_ITERATIONS=75
      - REQUEST_TIMEOUT=900
      - CONTEXT_LIMIT=8
      - SESSION_MAX_AGE_HOURS=48
      - CLEANUP_INTERVAL_MINUTES=120
      - LOG_LEVEL=info
      - LOG_TRUNCATE_LENGTH=400
      - DEBUG_MODE=false
```

## Configuration Validation

The system validates all configuration values:

- Numeric values must be positive integers
- Invalid values fall back to defaults
- Configuration is logged at startup for verification
- All settings are applied consistently across the application

## Performance Recommendations

For production environments:

- `REQUEST_TIMEOUT`: 300-600 seconds
- `MAX_ITERATIONS`: 50-100 (lower for faster responses)
- `CONTEXT_LIMIT`: 5-15 (higher uses more memory but provides better context)
- `SESSION_MAX_AGE_HOURS`: 12-48 hours
- `CLEANUP_INTERVAL_MINUTES`: 30-120 minutes
- `LOG_LEVEL`: `info` or `warn` (avoid `debug` in production)
- `LOG_TRUNCATE_LENGTH`: 200-500 characters

For development:

- `LOG_LEVEL`: `debug`
- `DEBUG_MODE`: `true`
- `LOG_TRUNCATE_LENGTH`: 500-1000 characters
- Lower timeouts for faster iteration 