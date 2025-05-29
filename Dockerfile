# Use Alpine Linux as base image since the app is designed for Alpine
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o skynet .

# Final stage
FROM alpine:latest

# Install system dependencies that the tools require
RUN apk add --no-cache \
    # Basic system tools
    coreutils \
    util-linux \
    procps \
    net-tools \
    iproute2 \
    iputils \
    bind-tools \
    # Process and system monitoring
    htop \
    iotop \
    lsof \
    strace \
    # File and text processing
    grep \
    sed \
    gawk \
    findutils \
    which \
    tree \
    less \
    # Network tools
    curl \
    wget \
    netcat-openbsd \
    tcpdump \
    nmap \
    # System administration
    shadow \
    sudo \
    # Docker client (for Docker tool)
    docker-cli \
    docker-cli-compose \
    # Package management (apk is already available)
    # Service management
    openrc \
    # Additional utilities
    bash \
    zsh \
    fish \
    nano \
    vim \
    # Time and timezone
    tzdata \
    # CA certificates
    ca-certificates \
    # Development tools that might be needed
    git \
    make \
    # Archive tools
    tar \
    gzip \
    unzip \
    # Permission and ownership tools
    acl

# Create necessary directories
RUN mkdir -p /app/static /tmp /var/log

# Set timezone
ENV TZ=UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Create a non-root user for security (though the app needs root privileges for system tools)
RUN addgroup -S skynet && adduser -S skynet -G skynet

# Copy the binary from builder stage
COPY --from=builder /app/skynet /app/skynet

# Copy static files
COPY --from=builder /app/static /app/static

# Set permissions
RUN chmod +x /app/skynet

# Create necessary directories and set permissions
RUN mkdir -p /var/run /var/log/skynet && \
    chown -R skynet:skynet /app && \
    chown -R skynet:skynet /var/log/skynet

# Set working directory
WORKDIR /app

# Environment variables with defaults
ENV PORT=8080 \
    LOG_LEVEL=info \
    OLLAMA_ENDPOINT=http://ollama:11434 \
    OLLAMA_MODEL=qwen3 \
    USER=root \
    HOME=/root

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/status || exit 1

# Switch to root user for system administration capabilities
USER root

# Start the application
CMD ["/app/skynet"] 