#!/bin/bash

# BackOffice E2E Test Runner
# This script starts the demo application and runs the E2E tests against it

set -e  # Exit on any error

# Default configuration
PORT=8080
HEADLESS=true
SLOW_MO="100ms"
TIMEOUT="1s"
DEMO_STARTUP_WAIT=1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

This script runs E2E tests for the BackOffice library from the e2e_testing directory.
It will automatically start the demo app, wait for it to be ready, run tests, and clean up.

OPTIONS:
    -p, --port PORT         Port number for demo app (default: 8080)
    -h, --headless BOOL     Run browser in headless mode (default: true)
    -s, --slow-mo DURATION  Slow down operations (default: 100ms)
    -t, --timeout DURATION  Timeout for operations (default: 30s)
    -w, --wait SECONDS      Wait time for demo startup (default: 5)
    --help                  Show this help message

EXAMPLES:
    $0                                          # Run with defaults
    $0 -p 3000 --headless false               # Run on port 3000 with visible browser
    $0 -w 10 -s 200ms                         # Wait 10s for startup, slow down operations

ENVIRONMENT VARIABLES:
    BACKOFFICE_E2E_PORT      Override default port
    BACKOFFICE_E2E_HEADLESS  Override headless mode (true/false)

NOTE:
    Run this script from the e2e_testing directory:
    cd e2e_testing && ./run_e2e_tests.sh
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -h|--headless)
            HEADLESS="$2"
            shift 2
            ;;
        -s|--slow-mo)
            SLOW_MO="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -w|--wait)
            DEMO_STARTUP_WAIT="$2"
            shift 2
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Override with environment variables if set
PORT=${BACKOFFICE_E2E_PORT:-$PORT}
HEADLESS=${BACKOFFICE_E2E_HEADLESS:-$HEADLESS}

# Validate port number
if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1024 ] || [ "$PORT" -gt 65535 ]; then
    print_error "Invalid port number: $PORT (must be between 1024-65535)"
    exit 1
fi

# Global variables for process management
DEMO_PID=""
DEMO_STARTED=false
DEMO_DIR="../examples/sql-example"  # Relative to e2e_testing directory

# Cleanup function
cleanup() {
    if [ ! -z "$DEMO_PID" ]; then
        print_status "Stopping demo application (PID: $DEMO_PID)..."
        # Try graceful shutdown first
        kill -TERM $DEMO_PID 2>/dev/null || true
        sleep 2
        # Force kill if still running
        if kill -0 $DEMO_PID 2>/dev/null; then
            print_status "Force killing demo application..."
            kill -KILL $DEMO_PID 2>/dev/null || true
        fi
        wait $DEMO_PID 2>/dev/null || true
        print_success "Demo application stopped"
        DEMO_PID=""
    fi
    
    # Only clean up port processes if we started a demo application
    if [ "$DEMO_STARTED" = true ]; then
        local port_pids=$(lsof -ti :$PORT 2>/dev/null || true)
        if [ ! -z "$port_pids" ]; then
            print_status "Cleaning up remaining processes on port $PORT..."
            echo $port_pids | xargs kill -KILL 2>/dev/null || true
        fi
    fi
}

# Set up signal handlers for cleanup
trap cleanup EXIT INT TERM


# Function to wait for demo app to be ready
wait_for_app() {
    local max_attempts=30
    local attempt=1
    
    print_status "Waiting for demo app to be ready on port $PORT..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "http://localhost:$PORT/admin/" > /dev/null 2>&1; then
            print_success "Demo app is ready!"
            return 0
        fi
        
        if [ $((attempt % 5)) -eq 0 ]; then
            print_status "Still waiting... (attempt $attempt/$max_attempts)"
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    print_error "Demo app failed to start within $max_attempts seconds"
    return 1
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in the e2e_testing directory
    if [ ! -f "go.mod" ] || [ ! -d "$DEMO_DIR" ]; then
        print_error "Please run this script from the e2e_testing directory"
        print_status "Expected: cd e2e_testing && ./run_e2e_tests.sh"
        print_status "Demo directory should be at: $DEMO_DIR"
        exit 1
    fi
    
    # Verify this is the e2e_testing module
    if ! grep -q "backoffice-e2e-testing" go.mod 2>/dev/null; then
        print_error "This doesn't appear to be the e2e_testing directory"
        print_status "Expected go.mod to contain 'backoffice-e2e-testing'"
        exit 1
    fi
    
    # Check if curl is available (for health check)
    if ! command -v curl &> /dev/null; then
        print_warning "curl not found, app readiness check will be skipped"
    fi
    
    print_success "Prerequisites check passed"
}

# Function to install Playwright if needed
install_playwright() {
    print_status "Checking Playwright Go installation..."
    
    # Check if playwright-go is in go.mod
    if ! grep -q "playwright-community/playwright-go" go.mod 2>/dev/null; then
        print_status "Installing Playwright Go dependency..."
        go mod edit -require=github.com/playwright-community/playwright-go@latest
        go mod tidy
    fi
    
    # Install playwright browsers if needed
    if ! go run github.com/playwright-community/playwright-go/cmd/playwright@latest list 2>/dev/null | grep -q chromium; then
        print_status "Installing Playwright browsers (this may take a while)..."
        go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
    fi
    
    print_success "Playwright is ready"
}

# Function to start demo application
start_demo() {
    print_status "Starting BackOffice demo application on port $PORT..."
    
    cd "$DEMO_DIR"
    
    # Build and start the demo app in background
    go run main.go > ../../e2e_testing/demo_output.log 2>&1 &
    DEMO_PID=$!
    DEMO_STARTED=true
    
    cd - > /dev/null
    
    print_status "Demo application started (PID: $DEMO_PID)"
    
    # Wait for the configured startup time
    print_status "Waiting ${DEMO_STARTUP_WAIT}s for demo application to initialize..."
    sleep $DEMO_STARTUP_WAIT
    
    # Check if process is still running
    if ! kill -0 $DEMO_PID 2>/dev/null; then
        print_error "Demo application failed to start"
        print_status "Check demo_output.log for details"
        cat "demo_output.log" 2>/dev/null || true
        exit 1
    fi
    
    # Wait for app to be ready (if curl is available)
    if command -v curl &> /dev/null; then
        if ! wait_for_app; then
            print_error "Demo application is not responding"
            print_status "Demo output:"
            cat "demo_output.log" 2>/dev/null || true
            exit 1
        fi
    fi
    
    print_success "Demo application is running at http://localhost:$PORT/admin/"
}

# Function to run E2E tests
run_tests() {
    print_status "Running E2E tests..."
    print_status "Configuration:"
    print_status "  Port: $PORT"
    print_status "  Headless: $HEADLESS"
    print_status "  Slow Motion: $SLOW_MO"
    print_status "  Timeout: $TIMEOUT"
    
    # Run the E2E tests
    go run e2e.go \
        -port="$PORT" \
        -headless="$HEADLESS" \
        -slow-mo="$SLOW_MO" \
        -timeout="$TIMEOUT"
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "All E2E tests passed! ✅"
    else
        print_error "E2E tests failed! ❌"
        print_status "Demo application logs:"
        cat "demo_output.log" 2>/dev/null || true
    fi
    
    return $exit_code
}

# Main execution
main() {
    print_status "BackOffice E2E Test Runner"
    print_status "=========================="
    print_status "Running from e2e_testing module (isolated dependencies)"
    
    check_prerequisites
    install_playwright
    
    # Check if port is available, and start demo only if needed
    if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        print_warning "Port $PORT is already in use - using existing service"
        print_status "Will run tests against whatever is running on port $PORT"
        print_status "Skipping demo application startup"
    else
        print_status "Port $PORT is available - starting demo application"
        start_demo
    fi
    
    # Run the tests
    if run_tests; then
        print_success "🎉 E2E test suite completed successfully!"
        cleanup  # Explicit cleanup on success
        exit 0
    else
        print_error "💥 E2E test suite failed!"
        cleanup  # Explicit cleanup on failure
        exit 1
    fi
}

# Run main function
main "$@"