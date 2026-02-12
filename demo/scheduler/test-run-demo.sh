#!/bin/bash
# test-run-demo.sh — Tests for run-demo.sh
# Validates dry-run output, argument parsing, and step sequencing.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/run-demo.sh"
PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

assert_contains() {
    local output="$1" expected="$2" label="$3"
    if echo "$output" | grep -qF "$expected"; then
        pass "$label"
    else
        fail "$label (expected to find: $expected)"
    fi
}

assert_not_contains() {
    local output="$1" unexpected="$2" label="$3"
    if echo "$output" | grep -qF "$unexpected"; then
        fail "$label (unexpectedly found: $unexpected)"
    else
        pass "$label"
    fi
}

assert_exit_code() {
    local actual="$1" expected="$2" label="$3"
    if [ "$actual" -eq "$expected" ]; then
        pass "$label"
    else
        fail "$label (expected exit $expected, got $actual)"
    fi
}

echo "=== run-demo.sh Test Suite ==="
echo ""

# ---------------------------------------------------------------
# Test 1: Script exists and is executable
# ---------------------------------------------------------------
echo "Test 1: Script exists and is executable"
if [ -x "$SCRIPT" ]; then
    pass "script is executable"
else
    fail "script is not executable or missing"
    echo ""
    echo "=== Results: $PASS passed, $FAIL failed ==="
    exit 1
fi
echo ""

# ---------------------------------------------------------------
# Test 2: --help shows usage
# ---------------------------------------------------------------
echo "Test 2: --help shows usage"
HELP_OUTPUT=$("$SCRIPT" --help 2>&1)
HELP_EXIT=$?
assert_exit_code "$HELP_EXIT" 0 "--help exits 0"
assert_contains "$HELP_OUTPUT" "Usage" "--help shows Usage"
assert_contains "$HELP_OUTPUT" "dry-run" "--help mentions dry-run"
assert_contains "$HELP_OUTPUT" "ORCHESTRATOR_URL" "--help mentions ORCHESTRATOR_URL"
echo ""

# ---------------------------------------------------------------
# Test 3: --dry-run outputs curl commands without executing
# ---------------------------------------------------------------
echo "Test 3: --dry-run outputs expected curl commands"
DRY_OUTPUT=$("$SCRIPT" --dry-run 2>&1)
DRY_EXIT=$?
assert_exit_code "$DRY_EXIT" 0 "--dry-run exits 0"
assert_contains "$DRY_OUTPUT" "curl" "--dry-run shows curl commands"
assert_contains "$DRY_OUTPUT" "tracking" "--dry-run mentions tracking"
assert_contains "$DRY_OUTPUT" "record" "--dry-run mentions record"
echo ""

# ---------------------------------------------------------------
# Test 4: Default URL is localhost:8080
# ---------------------------------------------------------------
echo "Test 4: Default orchestrator URL"
DRY_OUTPUT=$("$SCRIPT" --dry-run 2>&1)
assert_contains "$DRY_OUTPUT" "localhost:8080" "default URL is localhost:8080"
echo ""

# ---------------------------------------------------------------
# Test 5: ORCHESTRATOR_URL env var overrides default
# ---------------------------------------------------------------
echo "Test 5: ORCHESTRATOR_URL override"
CUSTOM_OUTPUT=$(ORCHESTRATOR_URL="http://myhost:9090" "$SCRIPT" --dry-run 2>&1)
assert_contains "$CUSTOM_OUTPUT" "myhost:9090" "custom URL is used"
assert_not_contains "$CUSTOM_OUTPUT" "localhost:8080" "default URL is not used"
echo ""

# ---------------------------------------------------------------
# Test 6: Duration argument is respected
# ---------------------------------------------------------------
echo "Test 6: Duration argument"
DUR_OUTPUT=$("$SCRIPT" --dry-run 5 2>&1)
assert_contains "$DUR_OUTPUT" "5 seconds" "duration 5 appears in output"

DUR_DEFAULT=$("$SCRIPT" --dry-run 2>&1)
assert_contains "$DUR_DEFAULT" "60 seconds" "default duration is 60"
echo ""

# ---------------------------------------------------------------
# Test 7: Correct step sequence
#   Expected order: enable tracking -> start recording -> wait -> stop recording -> disable tracking
# ---------------------------------------------------------------
echo "Test 7: Correct step sequence"
SEQ_OUTPUT=$("$SCRIPT" --dry-run 2>&1)

# Extract step lines and verify order using line numbers (case-insensitive)
STEP_TRACK_ON=$(echo "$SEQ_OUTPUT" | grep -in "enable.*tracking\|tracking.*enable" | head -1 | cut -d: -f1)
STEP_REC_START=$(echo "$SEQ_OUTPUT" | grep -in "start.*record\|record.*start" | head -1 | cut -d: -f1)
STEP_WAIT=$(echo "$SEQ_OUTPUT" | grep -in "[Ww]ait" | head -1 | cut -d: -f1)
STEP_REC_STOP=$(echo "$SEQ_OUTPUT" | grep -in "stop.*record\|record.*stop" | head -1 | cut -d: -f1)
STEP_TRACK_OFF=$(echo "$SEQ_OUTPUT" | grep -in "disable.*tracking\|tracking.*disable" | head -1 | cut -d: -f1)

if [ -n "$STEP_TRACK_ON" ] && [ -n "$STEP_REC_START" ] && [ -n "$STEP_WAIT" ] && \
   [ -n "$STEP_REC_STOP" ] && [ -n "$STEP_TRACK_OFF" ]; then
    if [ "$STEP_TRACK_ON" -lt "$STEP_REC_START" ] && \
       [ "$STEP_REC_START" -lt "$STEP_WAIT" ] && \
       [ "$STEP_WAIT" -lt "$STEP_REC_STOP" ] && \
       [ "$STEP_REC_STOP" -lt "$STEP_TRACK_OFF" ]; then
        pass "steps are in correct order"
    else
        fail "steps are out of order (track_on=$STEP_TRACK_ON rec_start=$STEP_REC_START wait=$STEP_WAIT rec_stop=$STEP_REC_STOP track_off=$STEP_TRACK_OFF)"
    fi
else
    fail "could not find all steps in output (track_on=$STEP_TRACK_ON rec_start=$STEP_REC_START wait=$STEP_WAIT rec_stop=$STEP_REC_STOP track_off=$STEP_TRACK_OFF)"
fi
echo ""

# ---------------------------------------------------------------
# Test 8: Uses correct orchestrator state API path
# ---------------------------------------------------------------
echo "Test 8: Uses state API path"
API_OUTPUT=$("$SCRIPT" --dry-run 2>&1)
assert_contains "$API_OUTPUT" "/api/systems/smart-room-demo/state" "uses /api/systems/{system}/state path"
assert_contains "$API_OUTPUT" "control_sets" "sends control_sets JSON body"
echo ""

# ---------------------------------------------------------------
# Test 9: Timestamped output
# ---------------------------------------------------------------
echo "Test 9: Timestamped output"
TS_OUTPUT=$("$SCRIPT" --dry-run 2>&1)
# Expect timestamps in format like [HH:MM:SS] or YYYY-MM-DD HH:MM:SS
if echo "$TS_OUTPUT" | grep -qE '\[[0-9]{2}:[0-9]{2}:[0-9]{2}\]|[0-9]{4}-[0-9]{2}-[0-9]{2}'; then
    pass "output includes timestamps"
else
    fail "no timestamps found in output"
fi
echo ""

# ---------------------------------------------------------------
# Summary
# ---------------------------------------------------------------
TOTAL=$((PASS + FAIL))
echo "=== Results: $PASS/$TOTAL passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
