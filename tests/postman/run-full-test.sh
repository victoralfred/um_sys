#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Running UManager API Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Newman is installed
if ! command -v newman &> /dev/null; then
    echo -e "${RED}Newman is not installed. Please install it first:${NC}"
    echo "npm install -g newman"
    exit 1
fi

# Run the tests
echo -e "${YELLOW}Starting test execution...${NC}"
echo ""

# Run tests and capture the exit code
newman run UManager_API_Tests.postman_collection.json \
    -e UManager_Local_Environment.postman_environment.json \
    --color on

EXIT_CODE=$?

echo ""
echo -e "${BLUE}========================================${NC}"

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}  ✅ ALL TESTS PASSED!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${GREEN}Test Summary:${NC}"
    echo "  • Health & API Info: ✅"
    echo "  • User Registration: ✅"
    echo "  • Authentication: ✅"
    echo "  • Protected Endpoints: ✅"
    echo "  • Token Management: ✅"
    echo "  • Logout: ✅"
    echo "  • Input Validation: ✅"
else
    echo -e "${RED}  ❌ SOME TESTS FAILED${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${RED}Please check the output above for details.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Your API is working correctly!${NC}"