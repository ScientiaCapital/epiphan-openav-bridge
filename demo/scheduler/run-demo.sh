#!/bin/bash
# run-demo.sh — One-shot smart room demo runner
#
# Orchestrates a timed recording session:
#   1. Enable EC20 AI tracking
#   2. Start Pearl recording
#   3. Wait for specified duration
#   4. Stop Pearl recording
#   5. Disable EC20 tracking
#
# Uses the OpenAV orchestrator state API:
#   PUT /api/systems/{system}/state  (partial JSON state tree)
#
# Environment:
#   ORCHESTRATOR_URL  Base URL of orchestrator (default: http://localhost:8080)
#
# Usage:
#   ./run-demo.sh [--dry-run] [--help] [duration_seconds]

set -uo pipefail

# ---------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------
ORCHESTRATOR_URL="${ORCHESTRATOR_URL:-http://localhost:8080}"
SYSTEM="smart-room-demo"
DRY_RUN=false
DURATION=60

# Track what we started so cleanup knows what to undo
TRACKING_ENABLED=false
RECORDING_STARTED=false

# ---------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------
for arg in "$@"; do
    case "$arg" in
        --help|-h)
            echo "Usage: $(basename "$0") [--dry-run] [--help] [duration_seconds]"
            echo ""
            echo "Run a timed smart room demo session."
            echo ""
            echo "Options:"
            echo "  --dry-run   Print curl commands without executing them"
            echo "  --help      Show this help message"
            echo ""
            echo "Arguments:"
            echo "  duration_seconds  Recording duration (default: 60)"
            echo ""
            echo "Environment:"
            echo "  ORCHESTRATOR_URL  Orchestrator base URL (default: http://localhost:8080)"
            exit 0
            ;;
        --dry-run)
            DRY_RUN=true
            ;;
        *)
            if [[ "$arg" =~ ^[0-9]+$ ]]; then
                DURATION="$arg"
            else
                echo "Unknown argument: $arg" >&2
                exit 1
            fi
            ;;
    esac
done

# ---------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------
timestamp() {
    date "+[%H:%M:%S]"
}

log() {
    echo "$(timestamp) $1"
}

state_url() {
    echo "${ORCHESTRATOR_URL}/api/systems/${SYSTEM}/state"
}

# Build a partial state JSON tree for a control_set/control/value
state_body() {
    local controlset="$1" control="$2" value="$3"
    printf '{"control_sets":{"%s":{"controls":{"%s":{"value":%s}}}}}' \
        "$controlset" "$control" "$value"
}

# Execute or print a PUT state request depending on mode
api_put() {
    local body="$1" description="$2"
    local url
    url="$(state_url)"
    if [ "$DRY_RUN" = true ]; then
        log "[DRY-RUN] $description"
        echo "  curl -X PUT \"$url\" -H \"Content-Type: application/json\" -d '$body'"
        return 0
    fi

    log "$description"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X PUT "$url" \
        -H "Content-Type: application/json" \
        -d "$body" 2>/dev/null)

    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        log "  OK (HTTP $http_code)"
        return 0
    else
        log "  FAIL (HTTP $http_code)"
        return 1
    fi
}

# ---------------------------------------------------------------
# Step functions
# ---------------------------------------------------------------
enable_tracking() {
    if api_put "$(state_body camera tracking true)" "Step 1: Enable EC20 AI tracking"; then
        TRACKING_ENABLED=true
    else
        return 1
    fi
}

start_recording() {
    if api_put "$(state_body recording record true)" "Step 2: Start Pearl recording"; then
        RECORDING_STARTED=true
    else
        return 1
    fi
}

wait_duration() {
    if [ "$DRY_RUN" = true ]; then
        log "[DRY-RUN] Step 3: Wait $DURATION seconds"
    else
        log "Step 3: Wait $DURATION seconds"
        sleep "$DURATION"
    fi
}

stop_recording() {
    api_put "$(state_body recording record false)" "Step 4: Stop Pearl recording"
}

disable_tracking() {
    api_put "$(state_body camera tracking false)" "Step 5: Disable EC20 tracking"
}

# ---------------------------------------------------------------
# Cleanup on failure — try to undo whatever we started
# ---------------------------------------------------------------
cleanup() {
    log "Cleaning up..."
    if [ "$RECORDING_STARTED" = true ]; then
        api_put "$(state_body recording record false)" "  Cleanup: stopping recording" || true
    fi
    if [ "$TRACKING_ENABLED" = true ]; then
        api_put "$(state_body camera tracking false)" "  Cleanup: disabling tracking" || true
    fi
}

# ---------------------------------------------------------------
# Main
# ---------------------------------------------------------------
main() {
    trap cleanup INT TERM
    echo "========================================="
    log "Smart Room Demo"
    log "Orchestrator: $ORCHESTRATOR_URL"
    log "Duration: $DURATION seconds"
    if [ "$DRY_RUN" = true ]; then
        log "Mode: dry-run"
    fi
    echo "========================================="
    echo ""

    if ! enable_tracking; then
        log "ABORT: Failed to enable tracking"
        cleanup
        exit 1
    fi
    echo ""

    if ! start_recording; then
        log "ABORT: Failed to start recording"
        cleanup
        exit 1
    fi
    echo ""

    wait_duration
    echo ""

    local exit_code=0

    if ! stop_recording; then
        log "WARNING: Failed to stop recording"
        exit_code=1
    fi
    echo ""

    if ! disable_tracking; then
        log "WARNING: Failed to disable tracking"
        exit_code=1
    fi
    echo ""

    echo "========================================="
    if [ "$exit_code" -eq 0 ]; then
        log "Demo complete — all steps succeeded"
    else
        log "Demo finished with errors"
    fi
    echo "========================================="
    return "$exit_code"
}

main
