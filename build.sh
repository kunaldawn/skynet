#!/bin/bash
set -e

# Skynet AI Agent Build and Deploy Script
# Usage: ./build.sh [action] [options]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
ACTION=""
USE_HOST_OLLAMA=true
PORT=8080
LOG_LEVEL=info
OLLAMA_MODEL=qwen3
OLLAMA_PORT=11434
LLM_PROVIDER=ollama
DETACHED=false
FORCE_REBUILD=false

# Print usage
usage() {
    echo "Skynet AI Agent Build and Deploy Script"
    echo ""
    echo "Usage: $0 [action] [options]"
    echo ""
    echo "Actions:"
    echo "  build         Build the Docker image"
    echo "  start         Start the services"
    echo "  stop          Stop the services"
    echo "  restart       Restart the services"
    echo "  logs          Show logs"
    echo "  status        Show service status"
    echo "  clean         Clean up containers and images"
    echo "  setup         Initialize .env file and check dependencies"
    echo ""
    echo "Options:"
    echo "  --host-ollama       Use host Ollama instead of containerized (default)"
    echo "  --container-ollama  Use containerized Ollama"
    echo "  --port PORT         Set web interface port (default: 8080)"
    echo "  --ollama-port PORT  Set Ollama port (default: 11434)"
    echo "  --log-level LVL     Set log level (debug|info|warn|error, default: info)"
    echo "  --model MODEL       Set Ollama model (default: qwen3)"
    echo "  --provider PROVIDER Set LLM provider (ollama|gemini, default: ollama)"
    echo "  --detached          Run in detached mode"
    echo "  --force-rebuild     Force rebuild of Docker image"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 setup                              # Initialize environment"
    echo "  $0 start --host-ollama --port 9090    # Start with host Ollama"
    echo "  $0 start --container-ollama           # Start with containerized Ollama"
    echo "  $0 build --force-rebuild              # Force rebuild Docker image"
    echo "  $0 logs --detached                    # Show logs without following"
    echo "  $0 restart --log-level debug          # Restart with debug logging"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        build|start|stop|restart|logs|status|clean|setup)
            ACTION="$1"
            shift
            ;;
        --host-ollama)
            USE_HOST_OLLAMA=true
            shift
            ;;
        --container-ollama)
            USE_HOST_OLLAMA=false
            shift
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --ollama-port)
            OLLAMA_PORT="$2"
            shift 2
            ;;
        --log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        --model)
            OLLAMA_MODEL="$2"
            shift 2
            ;;
        --provider)
            LLM_PROVIDER="$2"
            shift 2
            ;;
        --detached)
            DETACHED=true
            shift
            ;;
        --force-rebuild)
            FORCE_REBUILD=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

# Check if action is provided
if [[ -z "$ACTION" ]]; then
    echo -e "${RED}Error: No action specified${NC}"
    usage
    exit 1
fi

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        if ! docker compose version &> /dev/null; then
            log_error "Docker Compose is not installed or not available"
            exit 1
        fi
    fi
    
    # Check if running on a system with Docker socket access
    if [[ ! -S /var/run/docker.sock ]]; then
        log_warn "Docker socket not found at /var/run/docker.sock"
        log_warn "Some system administration features may not work properly"
    fi
    
    log_info "Prerequisites check passed"
}

# Create .env file if it doesn't exist
setup_env() {
    if [[ ! -f .env ]]; then
        log_info "Creating .env file from template..."
        if [[ -f .env.example ]]; then
            cp .env.example .env
        else
            log_warn ".env.example not found, creating basic .env file"
            cat > .env << EOF
# Skynet AI Agent Environment Configuration
PORT=${PORT}
LOG_LEVEL=${LOG_LEVEL}
OLLAMA_MODEL=${OLLAMA_MODEL}
OLLAMA_PORT=${OLLAMA_PORT}
LLM_PROVIDER=${LLM_PROVIDER}
OLLAMA_ENDPOINT=http://ollama:11434
EOF
        fi
    fi
    
    # Update environment variables
    log_info "Updating environment variables..."
    
    # Use sed to update values in .env file
    sed -i "s/^PORT=.*/PORT=${PORT}/" .env
    sed -i "s/^LOG_LEVEL=.*/LOG_LEVEL=${LOG_LEVEL}/" .env
    sed -i "s/^OLLAMA_MODEL=.*/OLLAMA_MODEL=${OLLAMA_MODEL}/" .env
    sed -i "s/^OLLAMA_PORT=.*/OLLAMA_PORT=${OLLAMA_PORT}/" .env
    sed -i "s/^LLM_PROVIDER=.*/LLM_PROVIDER=${LLM_PROVIDER}/" .env
    
    # Update Ollama endpoint to use containerized Ollama
    sed -i "s|^OLLAMA_ENDPOINT=.*|OLLAMA_ENDPOINT=http://ollama:11434|" .env
    
    # Add missing variables if they don't exist
    if ! grep -q "^LLM_PROVIDER=" .env; then
        echo "LLM_PROVIDER=${LLM_PROVIDER}" >> .env
    fi
}

# Build Docker image
build_image() {
    log_info "Building Skynet Docker image..."
    
    local build_args=""
    if [[ "$FORCE_REBUILD" == true ]]; then
        build_args="--no-cache"
    fi
    
    docker build ${build_args} -t skynet:latest .
    log_info "Build completed successfully"
}

# Start services
start_services() {
    setup_env
    
    local services=""
    local compose_args=""
    
    if [[ "$USE_HOST_OLLAMA" == true ]]; then
        log_info "Starting with host Ollama configuration..."
        services="skynet"
        
        # Check if host Ollama is running
        if ! curl -s http://localhost:${OLLAMA_PORT}/api/version > /dev/null; then
            log_warn "Host Ollama doesn't seem to be running on port ${OLLAMA_PORT}"
            log_warn "Make sure to start Ollama with: ollama serve"
            log_warn "Or use --container-ollama to run Ollama in a container"
        fi
    else
        log_info "Starting with containerized Ollama..."
        services="ollama skynet"
    fi
    
    # Determine run mode
    if [[ "$DETACHED" == true ]]; then
        compose_args="-d"
    fi
    
    log_info "Starting services on port ${PORT}..."
    log_info "LLM Provider: ${LLM_PROVIDER}"
    log_info "Ollama Model: ${OLLAMA_MODEL}"
    
    docker compose up ${compose_args} ${services}
}

# Stop services
stop_services() {
    log_info "Stopping services..."
    docker compose down
}

# Restart services
restart_services() {
    log_info "Restarting services..."
    stop_services
    start_services
}

# Show logs
show_logs() {
    local follow_flag=""
    if [[ "$DETACHED" == false ]]; then
        follow_flag="-f"
    fi
    
    docker compose logs ${follow_flag}
}

# Show status
show_status() {
    log_info "Service status:"
    docker compose ps
    
    echo ""
    log_info "Health checks:"
    
    # Check Skynet health
    if curl -s http://localhost:${PORT}/status > /dev/null; then
        echo -e "  Skynet API: ${GREEN}✓ Running${NC} (http://localhost:${PORT})"
    else
        echo -e "  Skynet API: ${RED}✗ Not accessible${NC}"
    fi
    
    # Check Ollama health
    local ollama_endpoint=""
    if [[ "$USE_HOST_OLLAMA" == true ]]; then
        ollama_endpoint="http://localhost:${OLLAMA_PORT}"
        if curl -s ${ollama_endpoint}/api/version > /dev/null; then
            echo -e "  Host Ollama: ${GREEN}✓ Running${NC} (${ollama_endpoint})"
        else
            echo -e "  Host Ollama: ${RED}✗ Not accessible${NC}"
        fi
    else
        ollama_endpoint="http://localhost:${OLLAMA_PORT}"
        if curl -s ${ollama_endpoint}/api/version > /dev/null; then
            echo -e "  Ollama Container: ${GREEN}✓ Running${NC} (${ollama_endpoint})"
        else
            echo -e "  Ollama Container: ${RED}✗ Not accessible${NC}"
        fi
    fi
    
    # Check Docker socket access
    if docker info > /dev/null 2>&1; then
        echo -e "  Docker Access: ${GREEN}✓ Available${NC}"
    else
        echo -e "  Docker Access: ${RED}✗ Not available${NC}"
    fi
}

# Clean up
clean_up() {
    log_warn "This will remove all Skynet containers, images, and volumes"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Stopping and removing containers..."
        docker compose down -v --remove-orphans
        
        log_info "Removing images..."
        docker rmi skynet:latest 2>/dev/null || true
        docker rmi $(docker images -q --filter "reference=*skynet*") 2>/dev/null || true
        
        # Clean up Ollama images if they exist
        docker rmi ollama/ollama:latest 2>/dev/null || true
        
        log_info "Cleanup completed"
    else
        log_info "Cleanup cancelled"
    fi
}

# Setup environment
setup_environment() {
    log_info "Setting up Skynet AI Agent environment..."
    
    check_prerequisites
    setup_env
    
    log_info "Environment setup completed"
    log_info "Configuration file: .env"
    log_info ""
    log_info "Next steps:"
    log_info "  1. Review and customize .env file if needed"
    log_info "  2. Run './build.sh build' to build the Docker image"
    log_info "  3. Run './build.sh start' to start the services"
}

# Main execution
main() {
    case $ACTION in
        setup)
            setup_environment
            ;;
        build)
            check_prerequisites
            build_image
            ;;
        start)
            check_prerequisites
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            check_prerequisites
            restart_services
            ;;
        logs)
            show_logs
            ;;
        status)
            show_status
            ;;
        clean)
            clean_up
            ;;
        *)
            log_error "Unknown action: $ACTION"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main 