#\!/bin/bash
echo "=== Tracing commit process ==="
echo "1. Current .git/COMMIT_EDITMSG content:"
cat .git/COMMIT_EDITMSG 2>/dev/null || echo "  File doesn't exist"

echo -e "\n2. Creating test commit message..."
echo "IMPL: Test commit message" > .git/COMMIT_EDITMSG

echo -e "\n3. New .git/COMMIT_EDITMSG content:"
cat .git/COMMIT_EDITMSG

echo -e "\n4. Running hook check..."
FIRST_LINE=$(head -1 .git/COMMIT_EDITMSG)
echo "  First line extracted: '$FIRST_LINE'"

if echo "$FIRST_LINE" | grep -qE "^(RED:|GREEN:|REFACTOR:|TEST:|IMPL:|INITIAL:|FIX:|DOCS:|DEPS:)"; then
    echo "  ✓ Message validates correctly"
else
    echo "  ✗ Message fails validation"
fi
