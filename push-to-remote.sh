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

# Push main branch
echo -e "${YELLOW}Pushing main branch...${NC}"
git push -u origin main

# Push staging branch
echo -e "${YELLOW}Pushing staging branch...${NC}"
git push -u origin staging

# Push all feature branches
echo -e "${YELLOW}Pushing feature branches...${NC}"
git push -u origin feature/database-schema-design
git push -u origin feature/user-repository-implementation
git push -u origin feature/jwt-authentication
git push -u origin feature/rbac-system
git push -u origin feature/mfa-support
git push -u origin feature/subscription-billing
git push -u origin feature/password-policies
git push -u origin feature/auth-implementation
git push -u origin feature/analytics-extensions
git push -u origin feature/background-jobs
git push -u origin feature/enterprise-features
git push -u origin feature/feature-flags
git push -u origin feature/job-configuration-api
git push -u origin feature/redis-session-management
git push -u origin feature/refresh-logout-handlers
git push -u origin feature/usage-analytics-metrics
git push -u origin feature/user-profile-management

echo -e "${GREEN}✓ All branches pushed successfully!${NC}"

# Remove token from remote URL for security
echo -e "${YELLOW}Cleaning up credentials from git config...${NC}"
git remote set-url origin https://github.com/victoralfred/um_sys.git

echo -e "${GREEN}✓ Done! Credentials removed from git config.${NC}"