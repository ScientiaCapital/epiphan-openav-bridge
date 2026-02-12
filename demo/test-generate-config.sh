#!/usr/bin/env bash
# test-generate-config.sh — Tests for generate-config.sh
# Runs RED/GREEN verification for the config generator.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/system-configs/smart-room-demo.json"
TEST_ENV_FILE="$SCRIPT_DIR/.env.test"
BACKUP_FILE="$CONFIG_FILE.bak"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# Ensure cleanup runs even on early exit
cleanup() {
    [ -f "$BACKUP_FILE" ] && mv "$BACKUP_FILE" "$CONFIG_FILE"
    rm -f "$TEST_ENV_FILE" "$SCRIPT_DIR/.env.partial" "$SCRIPT_DIR/.env.metachar"
}
trap cleanup EXIT

# --- Setup ---
echo "=== Setup ==="

# Back up original config
cp "$CONFIG_FILE" "$BACKUP_FILE"

# Create test .env file with known values
cat > "$TEST_ENV_FILE" <<'TESTENV'
PEARL_HOST=10.0.0.50
PEARL_USERNAME=myadmin
PEARL_PASSWORD=s3cret!
EC20_HOST=10.0.0.51
EC20_USERNAME=camuser
EC20_PASSWORD=c@mpa$$
TESTENV

echo "Test .env created, original config backed up."
echo ""

# --- Test: generate-config.sh exists and is executable ---
echo "=== Test: Script exists and is executable ==="
if [ -x "$SCRIPT_DIR/generate-config.sh" ]; then
    pass "generate-config.sh is executable"
else
    fail "generate-config.sh does not exist or is not executable"
fi

# --- Test: Script runs successfully ---
echo "=== Test: Script runs successfully ==="
if bash "$SCRIPT_DIR/generate-config.sh" "$TEST_ENV_FILE" 2>/dev/null; then
    pass "generate-config.sh exited 0"
else
    fail "generate-config.sh exited non-zero"
fi

# --- Test: Output is valid JSON ---
echo "=== Test: Output is valid JSON ==="
# Use python (available on macOS) to validate JSON since jq may not be installed
if python3 -c "import json, sys; json.load(sys.stdin)" < "$CONFIG_FILE" 2>/dev/null; then
    pass "Output is valid JSON"
else
    fail "Output is NOT valid JSON"
fi

# --- Test: Pearl credentials substituted ---
echo "=== Test: Pearl credentials substituted ==="
if grep -q "myadmin:s3cret!@10.0.0.50" "$CONFIG_FILE"; then
    pass "Pearl host/username/password substituted correctly"
else
    fail "Pearl credentials not found in output"
fi

# --- Test: EC20 credentials substituted ---
echo "=== Test: EC20 credentials substituted ==="
if grep -q 'camuser:c@mpa$$@10.0.0.51' "$CONFIG_FILE"; then
    pass "EC20 host/username/password substituted correctly"
else
    fail "EC20 credentials not found in output"
fi

# --- Test: Placeholder values are gone ---
echo "=== Test: No placeholder values remain ==="
PLACEHOLDERS_FOUND=0
if grep -q "admin:password@pearl-host" "$CONFIG_FILE"; then
    fail "Placeholder 'admin:password@pearl-host' still present"
    PLACEHOLDERS_FOUND=1
fi
if grep -q "admin:password@ec20-host" "$CONFIG_FILE"; then
    fail "Placeholder 'admin:password@ec20-host' still present"
    PLACEHOLDERS_FOUND=1
fi
if [ "$PLACEHOLDERS_FOUND" -eq 0 ]; then
    pass "No placeholder values remain"
fi

# --- Test: system_name preserved ---
echo "=== Test: system_name preserved ==="
if grep -q '"system_name": "Smart Room Demo"' "$CONFIG_FILE"; then
    pass "system_name field preserved"
else
    fail "system_name field missing or changed"
fi

# --- Test: Idempotent (run twice, same result) ---
echo "=== Test: Idempotent ==="
FIRST_HASH=$(shasum "$CONFIG_FILE" | awk '{print $1}')
bash "$SCRIPT_DIR/generate-config.sh" "$TEST_ENV_FILE" 2>/dev/null || true
SECOND_HASH=$(shasum "$CONFIG_FILE" | awk '{print $1}')
if [ "$FIRST_HASH" = "$SECOND_HASH" ]; then
    pass "Idempotent — same output on second run"
else
    fail "NOT idempotent — output changed on second run"
fi

# --- Test: Passwords with sed metacharacters (& | \) ---
echo "=== Test: Sed metacharacters in password ==="
METACHAR_ENV="$SCRIPT_DIR/.env.metachar"
cat > "$METACHAR_ENV" <<'METAENV'
PEARL_HOST=10.0.0.60
PEARL_USERNAME=admin
PEARL_PASSWORD=p&ss|w\rd
EC20_HOST=10.0.0.61
EC20_USERNAME=admin
EC20_PASSWORD=t&st|p\ss
METAENV
if bash "$SCRIPT_DIR/generate-config.sh" "$METACHAR_ENV" 2>/dev/null; then
    # Verify the literal metacharacters appear in output
    if grep -qF 'admin:p&ss|w\rd@10.0.0.60' "$CONFIG_FILE" && \
       grep -qF 'admin:t&st|p\ss@10.0.0.61' "$CONFIG_FILE"; then
        pass "Sed metacharacters preserved in passwords"
    else
        fail "Sed metacharacters corrupted in output"
    fi
else
    fail "Script failed with sed metacharacters in password"
fi
rm -f "$METACHAR_ENV"
# Regenerate with original test env for remaining tests
bash "$SCRIPT_DIR/generate-config.sh" "$TEST_ENV_FILE" 2>/dev/null || true

# --- Test: Missing env var causes error ---
echo "=== Test: Missing env var causes error ==="
PARTIAL_ENV="$SCRIPT_DIR/.env.partial"
cat > "$PARTIAL_ENV" <<'PARTIALENV'
PEARL_HOST=10.0.0.50
PARTIALENV
if bash "$SCRIPT_DIR/generate-config.sh" "$PARTIAL_ENV" 2>/dev/null; then
    fail "Script should fail when required env vars are missing"
else
    pass "Script correctly fails on missing env vars"
fi
rm -f "$PARTIAL_ENV"

# --- Teardown (trap handles backup restore + temp file removal) ---
echo ""
echo "=== Teardown ==="
echo "Original config restored, test files cleaned up."

# --- Summary ---
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo ""
if [ "$FAIL" -gt 0 ]; then
    echo "TESTS FAILED"
    exit 1
else
    echo "ALL TESTS PASSED"
    exit 0
fi
