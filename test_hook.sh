#\!/bin/bash
echo "Testing commit message validation..."

# Function to check commit message
check_commit_message() {
    local msg="$1"
    echo "Checking message: '$msg'"
    if echo "$msg" | grep -qE "^(RED:|GREEN:|REFACTOR:|TEST:|IMPL:|INITIAL:|FIX:|DOCS:|DEPS:)"; then
        echo "✓ Message follows TDD convention"
        return 0
    else
        echo "✗ Message doesn't follow TDD convention"
        return 1
    fi
}

# Test various messages
check_commit_message "IMPL: Test message"
check_commit_message "TEST: Another test"
check_commit_message "bad message"
check_commit_message "feat: bad prefix"
