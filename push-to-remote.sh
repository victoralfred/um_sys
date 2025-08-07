#!/bin/bash

# Script to push all branches to remote repository using credentials
# This script reads credentials from .env.credentials file

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env.credentials exists
if [ ! -f .env.credentials ]; then
    echo -e "${RED}Error: .env.credentials file not found!${NC}"
    echo "Please create .env.credentials file with your GitHub token"
    exit 1
fi

# Source the credentials file
source .env.credentials

# Check if token is set
if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}Error: GITHUB_TOKEN not set in .env.credentials!${NC}"
    echo "Please add your GitHub Personal Access Token to .env.credentials"
    echo "Generate a token at: https://github.com/settings/tokens"
    exit 1
fi

echo -e "${YELLOW}Setting up remote with authentication...${NC}"

# Set the remote URL with token
git remote set-url origin https://${GITHUB_TOKEN}@github.com/victoralfred/um_sys.git

echo -e "${GREEN}Remote configured successfully${NC}"
echo -e "${YELLOW}Pushing branches to remote...${NC}"

# Get all local branches
echo -e "${YELLOW}Discovering local branches...${NC}"
BRANCHES=$(git branch --format='%(refname:short)')

# Push each branch
for branch in $BRANCHES; do
    echo -e "${YELLOW}Pushing branch: $branch...${NC}"
    if git push -u origin "$branch" 2>/dev/null; then
        echo -e "${GREEN}✓ Successfully pushed $branch${NC}"
    else
        echo -e "${RED}✗ Failed to push $branch (may not exist locally or has no changes)${NC}"
    fi
done

echo -e "${GREEN}✓ All branches pushed successfully!${NC}"

# Remove token from remote URL for security
echo -e "${YELLOW}Cleaning up credentials from git config...${NC}"
git remote set-url origin https://github.com/victoralfred/um_sys.git

echo -e "${GREEN}✓ Done! Credentials removed from git config.${NC}"