# Git Hook Fix Documentation

## Issue
The git pre-commit hook was rejecting valid commit messages that followed the TDD naming convention (RED:, GREEN:, REFACTOR:, TEST:, IMPL:, INITIAL:, FIX:, DOCS:, DEPS:).

## Root Cause
The pre-commit hook was trying to validate commit messages by reading from `.git/COMMIT_EDITMSG`, but this file is not yet created when using `git commit -m` during the pre-commit phase. The file is only available during the `prepare-commit-msg` phase.

## Solution

### 1. Created `prepare-commit-msg` Hook
Created a new hook at `.git/hooks/prepare-commit-msg` that validates commit messages at the correct time in the git workflow:

```bash
#!/bin/bash

# This hook is invoked by git commit right after preparing the default log message,
# and before the editor is started.

# Get the commit message file
COMMIT_MSG_FILE=$1
COMMIT_SOURCE=$2
SHA1=$3

# Read the first line of the commit message
if [ -f "$COMMIT_MSG_FILE" ]; then
    FIRST_LINE=$(head -1 "$COMMIT_MSG_FILE")
    
    # Check if it follows TDD convention
    if ! echo "$FIRST_LINE" | grep -qE "^(RED:|GREEN:|REFACTOR:|TEST:|IMPL:|INITIAL:|FIX:|DOCS:|DEPS:)"; then
        echo "❌ Commit doesn't follow TDD naming convention" >&2
        echo "Use one of: RED:, GREEN:, REFACTOR:, TEST:, IMPL:, INITIAL:, FIX:, DOCS:, DEPS: prefix" >&2
        exit 1
    fi
fi

exit 0
```

### 2. Modified Pre-commit Hook
Removed commit message validation from the pre-commit hook since it now focuses solely on code quality checks (formatting, linting, tests).

## Key Learnings

1. **Git Hook Lifecycle**: 
   - `pre-commit`: Runs before the commit message is prepared. Good for code quality checks.
   - `prepare-commit-msg`: Runs after the commit message is prepared but before the editor opens. Ideal for message validation.
   - `commit-msg`: Runs after the user has entered a commit message. Alternative place for message validation.

2. **Proper Hook Selection**: Choose the right hook for the right job. Message validation should happen in `prepare-commit-msg` or `commit-msg`, not `pre-commit`.

3. **Shell Script Escaping**: Be careful with special characters in shell scripts. The original hook had `\!` which caused syntax errors.

## Testing
After implementing the fix, commits with proper TDD prefixes work correctly:
- `git commit -m "FIX: Apply linter suggestions"` ✓
- `git commit -m "IMPL: Add new feature"` ✓
- `git commit -m "bad message"` ✗ (rejected as expected)