#!/bin/bash

# UManager API Postman Tests Runner
# This script runs the Postman collection tests using Newman

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COLLECTION_FILE="UManager_API_Collection.postman_collection.json"
ENVIRONMENT_FILE="UManager_Local_Environment.postman_environment.json"
BASE_URL="${BASE_URL:-http://localhost:8080}"
ITERATIONS="${ITERATIONS:-1}"
DELAY="${DELAY:-100}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  UManager API Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Newman is installed
if ! command -v newman &> /dev/null; then
    echo -e "${YELLOW}Newman is not installed. Installing...${NC}"
    npm install -g newman
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to install Newman. Please install it manually:${NC}"
        echo "npm install -g newman"
        exit 1
    fi
fi

# Check if newman-reporter-htmlextra is installed for HTML reports
if ! npm list -g newman-reporter-htmlextra &> /dev/null; then
    echo -e "${YELLOW}Installing HTML reporter for better test reports...${NC}"
    npm install -g newman-reporter-htmlextra
fi

# Function to run tests
run_tests() {
    local test_type=$1
    local folder=$2
    
    echo -e "${YELLOW}Running $test_type tests...${NC}"
    
    if [ -z "$folder" ]; then
        # Run entire collection
        newman run "$COLLECTION_FILE" \
            -e "$ENVIRONMENT_FILE" \
            --global-var "baseUrl=$BASE_URL" \
            -n "$ITERATIONS" \
            --delay-request "$DELAY" \
            --reporters cli,htmlextra \
            --reporter-htmlextra-export "test-results-$(date +%Y%m%d-%H%M%S).html" \
            --reporter-htmlextra-title "UManager API Test Results" \
            --reporter-htmlextra-darkTheme
    else
        # Run specific folder
        newman run "$COLLECTION_FILE" \
            -e "$ENVIRONMENT_FILE" \
            --folder "$folder" \
            --global-var "baseUrl=$BASE_URL" \
            -n "$ITERATIONS" \
            --delay-request "$DELAY" \
            --reporters cli,htmlextra \
            --reporter-htmlextra-export "test-results-$folder-$(date +%Y%m%d-%H%M%S).html" \
            --reporter-htmlextra-title "UManager API Test Results - $folder" \
            --reporter-htmlextra-darkTheme
    fi
    
    return $?
}

# Function to run load tests
run_load_tests() {
    echo -e "${YELLOW}Running load tests (10 iterations with 100ms delay)...${NC}"
    
    newman run "$COLLECTION_FILE" \
        -e "$ENVIRONMENT_FILE" \
        --folder "Load Testing" \
        --global-var "baseUrl=$BASE_URL" \
        -n 10 \
        --delay-request 100 \
        --reporters cli
    
    return $?
}

# Main menu
show_menu() {
    echo -e "${GREEN}Select test suite to run:${NC}"
    echo "1) All Tests"
    echo "2) Health & Info Tests"
    echo "3) Authentication Tests"
    echo "4) Protected Endpoints Tests"
    echo "5) Load Tests"
    echo "6) Custom (specify iterations and delay)"
    echo "7) Exit"
    echo ""
    read -p "Enter choice [1-7]: " choice
    
    case $choice in
        1)
            run_tests "all" ""
            ;;
        2)
            run_tests "Health & Info" "Health & Info"
            ;;
        3)
            run_tests "Authentication" "Authentication"
            ;;
        4)
            run_tests "Protected Endpoints" "Protected Endpoints"
            ;;
        5)
            run_load_tests
            ;;
        6)
            read -p "Enter number of iterations: " ITERATIONS
            read -p "Enter delay between requests (ms): " DELAY
            run_tests "custom" ""
            ;;
        7)
            echo -e "${GREEN}Goodbye!${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            show_menu
            ;;
    esac
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --all)
            run_tests "all" ""
            exit $?
            ;;
        --auth)
            run_tests "Authentication" "Authentication"
            exit $?
            ;;
        --health)
            run_tests "Health & Info" "Health & Info"
            exit $?
            ;;
        --protected)
            run_tests "Protected Endpoints" "Protected Endpoints"
            exit $?
            ;;
        --load)
            run_load_tests
            exit $?
            ;;
        --url)
            BASE_URL="$2"
            shift 2
            ;;
        --iterations)
            ITERATIONS="$2"
            shift 2
            ;;
        --delay)
            DELAY="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --all              Run all tests"
            echo "  --auth             Run authentication tests only"
            echo "  --health           Run health check tests only"
            echo "  --protected        Run protected endpoint tests only"
            echo "  --load             Run load tests"
            echo "  --url URL          Set base URL (default: http://localhost:8080)"
            echo "  --iterations N     Set number of iterations (default: 1)"
            echo "  --delay MS         Set delay between requests in ms (default: 0)"
            echo "  --help             Show this help message"
            echo ""
            echo "If no options are provided, an interactive menu will be shown."
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# If no arguments provided, show interactive menu
if [ $# -eq 0 ]; then
    show_menu
fi

# Check test results
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  ✓ Tests completed successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}  ✗ Some tests failed!${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi