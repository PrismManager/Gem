#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Root directory of the project
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Help message
function show_help {
    echo -e "${YELLOW}Gem Development Script${NC}"
    echo "Usage: ./scripts/dev.sh [command]"
    echo ""
    echo "Commands:"
    echo "  build       - Build the Gem binary"
    echo "  test        - Run tests"
    echo "  lint        - Run linter"
    echo "  clean       - Clean build artifacts"
    echo "  install     - Install Gem to /usr/local/bin"
    echo "  uninstall   - Uninstall Gem from /usr/local/bin"
    echo "  run         - Build and run Gem"
    echo "  help        - Show this help message"
}

# Build the binary
function build {
    echo -e "${GREEN}Building Gem...${NC}"
    go build -o gem
    echo -e "${GREEN}Build complete: $(pwd)/gem${NC}"
}

# Run tests
function run_tests {
    echo -e "${GREEN}Running tests...${NC}"
    go test -v ./...
}

# Run linter
function run_lint {
    echo -e "${GREEN}Running linter...${NC}"
    if ! command -v golangci-lint &> /dev/null; then
        echo -e "${YELLOW}golangci-lint not found, installing...${NC}"
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.54.2
    fi
    golangci-lint run ./...
}

# Clean build artifacts
function clean {
    echo -e "${GREEN}Cleaning build artifacts...${NC}"
    rm -f gem
    go clean
    echo -e "${GREEN}Clean complete${NC}"
}

# Install Gem
function install {
    echo -e "${GREEN}Installing Gem...${NC}"
    build
    sudo cp gem /usr/local/bin/
    echo -e "${GREEN}Installed Gem to /usr/local/bin/gem${NC}"
}

# Uninstall Gem
function uninstall {
    echo -e "${GREEN}Uninstalling Gem...${NC}"
    sudo rm -f /usr/local/bin/gem
    echo -e "${GREEN}Uninstalled Gem from /usr/local/bin/gem${NC}"
}

# Build and run
function run {
    echo -e "${GREEN}Building and running Gem...${NC}"
    build
    echo -e "${GREEN}Running Gem...${NC}"
    ./gem "$@"
}

# Main
if [[ $# -eq 0 ]]; then
    show_help
    exit 0
fi

case "$1" in
    build)
        build
        ;;
    test)
        run_tests
        ;;
    lint)
        run_lint
        ;;
    clean)
        clean
        ;;
    install)
        install
        ;;
    uninstall)
        uninstall
        ;;
    run)
        shift
        run "$@"
        ;;
    help)
        show_help
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        show_help
        exit 1
        ;;
esac
