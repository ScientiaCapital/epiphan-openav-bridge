#!/usr/bin/env bash
#
# ec20_probe.sh — EC20 REST API endpoint discovery harness
#
# Purpose: the 12 REST endpoint paths in source/driver.go are PLACEHOLDER values because
# the EC20's REST API paths are not published in Epiphan's public docs (see
# .claude/programs/ec20-api-discovery.md). This script operationalizes that discovery
# program so confirming the real paths on hardware is a ~30-minute job, not a research
# project.
#
# For each endpoint it probes the current placeholder plus a list of candidate paths, one
# at a time, and reports which return a "path exists" signal. It then prints a summary
# mapping each driver.go constant to the confirmed path, ready to paste back in.
#
# SAFETY: probes use GET only (never POST). Firing discovery POSTs at /ptz/pan or
# /preset/save would physically move the camera or overwrite presets. A 405 (Method Not
# Allowed) still confirms a path exists without mutating anything, so discovery stays
# non-destructive.
#
# Usage:
#   EC20_HOST=192.168.1.88 EC20_USERNAME=admin EC20_PASSWORD=secret bash ec20_probe.sh
#
# Env:
#   EC20_HOST            (required) camera IP/hostname — NO hardcoded creds in this file
#   EC20_USERNAME        default: admin
#   EC20_PASSWORD        default: (empty)
#   EC20_PORT            default: 80   (DOC-CONFIRMED HTTP port; 443 disabled)
#   EC20_PROBE_TIMEOUT   default: 30   (seconds per probe, per discovery program)
#   EC20_DRY_RUN=1       skip all network calls; just print the probe plan + summary
#
set -uo pipefail

HOST="${EC20_HOST:-}"
USERNAME="${EC20_USERNAME:-admin}"
PASSWORD="${EC20_PASSWORD:-}"
PORT="${EC20_PORT:-80}"
TIMEOUT="${EC20_PROBE_TIMEOUT:-30}"
DRY_RUN="${EC20_DRY_RUN:-0}"

if [[ -z "$HOST" && "$DRY_RUN" != "1" ]]; then
  echo "ERROR: EC20_HOST is required."
  echo "Usage: EC20_HOST=192.168.x.x EC20_USERNAME=admin EC20_PASSWORD=... bash ec20_probe.sh"
  echo "       (or EC20_DRY_RUN=1 bash ec20_probe.sh to preview without hardware)"
  exit 1
fi

BASE="http://${HOST:-DRYRUN}:${PORT}"
LOG="ec20_probe_$(date +%Y%m%d_%H%M%S).log"
declare -a RESULTS=()

log() { echo "$@" | tee -a "$LOG"; }

# probe LABEL PATH...  → prints the http code + verdict for each candidate path
probe() {
  local label="$1"; shift
  log ""
  log "── ${label} ──"
  local found=""
  for path in "$@"; do
    local code verdict
    if [[ "$DRY_RUN" == "1" ]]; then
      code="DRY"; verdict="(dry-run: would GET ${BASE}${path})"
    else
      # --digest: the real EC20 (lighttpd) requires HTTP Digest auth, not Basic.
      # NOTE: this device returns HTTP 200 with body {"err":"Invalid API command"}
      # for unknown /api/ commands, so a 200 alone is NOT proof — inspect the body.
      code=$(curl -s -o /dev/null -w '%{http_code}' \
        --max-time "$TIMEOUT" --digest -u "${USERNAME}:${PASSWORD}" \
        -X GET "${BASE}${path}" 2>/dev/null)
      case "$code" in
        200) verdict="CONFIRMED? (GET 200 — verify body is not an error)"; [[ -z "$found" ]] && found="$path" ;;
        401) verdict="AUTH (path likely exists — check credentials)" ;;
        405) verdict="PATH EXISTS (405 — likely POST-only)"; [[ -z "$found" ]] && found="${path} (POST?)" ;;
        404) verdict="not found" ;;
        000) verdict="UNREACHABLE / timeout" ;;
        *)   verdict="HTTP ${code}" ;;
      esac
    fi
    printf '  %-4s  %-42s  →  %s\n' "$code" "$path" "$verdict" | tee -a "$LOG"
    [[ "$DRY_RUN" == "1" ]] || sleep 1
  done
  RESULTS+=("${label}|${found:-NEEDS-PROBE (no candidate matched)}")
}

log "EC20 REST API discovery probe"
log "Target: ${BASE}   user: ${USERNAME}   timeout: ${TIMEOUT}s   dry_run: ${DRY_RUN}"
log "Log file: ${LOG}"

# ---- Preflight reachability (fast) so an unreachable host fails clean, not after 12×30s ----
if [[ "$DRY_RUN" != "1" ]]; then
  pre=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 --digest -u "${USERNAME}:${PASSWORD}" "${BASE}/" 2>/dev/null)
  if [[ "$pre" == "000" ]]; then
    log ""
    log "!! ${HOST}:${PORT} is unreachable (preflight timed out). Skipping probes."
    log "   Confirm the EC20 is on the network and EC20_HOST is correct, then re-run."
    DRY_RUN="skip"
  else
    log "Preflight: reached ${BASE}/ (HTTP ${pre})"
  fi
fi

# ---- First move: does the camera expose an API index or docs? (per discovery program) ----
if [[ "$DRY_RUN" != "skip" ]]; then
  probe "API-index (first move)" "/api/" "/api/v1/" "/swagger.json" "/openapi.json" "/api/docs"
fi

# ---- The 12 endpoints (candidates mirror .claude/programs/ec20-api-discovery.md) ----
if [[ "$DRY_RUN" != "skip" ]]; then
  probe "ec20EndpointStatus"      "/api/status" "/api/v1/status" "/status" "/api/device/status" "/api/info" "/api/system/status" "/cgi-bin/status"
  probe "ec20EndpointPosition"    "/api/ptz/position" "/api/v1/ptz/position" "/ptz/position" "/api/ptz/query" "/api/ptz/status" "/cgi-bin/ptz.cgi?action=getStatus"
  probe "ec20EndpointPan"         "/api/ptz/pan" "/api/v1/ptz/pan" "/api/ptz/move" "/api/ptz/continuous" "/cgi-bin/ptz.cgi?action=pan"
  probe "ec20EndpointTilt"        "/api/ptz/tilt" "/api/v1/ptz/tilt" "/api/ptz/move" "/api/ptz/continuous" "/cgi-bin/ptz.cgi?action=tilt"
  probe "ec20EndpointZoom"        "/api/ptz/zoom" "/api/v1/ptz/zoom" "/api/ptz/zoom/set" "/api/zoom" "/cgi-bin/ptz.cgi?action=zoom"
  probe "ec20EndpointHome"        "/api/ptz/home" "/api/v1/ptz/home" "/api/ptz/preset/home" "/api/ptz/goto/home" "/cgi-bin/ptz.cgi?action=home"
  probe "ec20EndpointPresets"     "/api/ptz/presets" "/api/v1/ptz/presets" "/api/presets" "/api/ptz/preset/list" "/cgi-bin/ptz.cgi?action=getPresets"
  probe "ec20EndpointPresetGoto"  "/api/ptz/preset/goto" "/api/v1/ptz/preset/goto" "/api/ptz/preset/call" "/api/ptz/goto" "/cgi-bin/ptz.cgi?action=gotoPreset"
  probe "ec20EndpointPresetSave"  "/api/ptz/preset/save" "/api/v1/ptz/preset/save" "/api/ptz/preset/set" "/api/ptz/preset/store" "/cgi-bin/ptz.cgi?action=setPreset"
  probe "ec20EndpointTrackingOn"  "/api/tracking/enable" "/api/v1/tracking/enable" "/api/tracking/start" "/api/tracking/on" "/api/ai/tracking/enable"
  probe "ec20EndpointTrackingOff" "/api/tracking/disable" "/api/v1/tracking/disable" "/api/tracking/stop" "/api/tracking/off" "/api/ai/tracking/disable"
  probe "ec20EndpointPreview"     "/api/preview" "/api/v1/preview" "/api/snapshot" "/api/image" "/cgi-bin/snapshot.cgi" "/api/capture" "/preview.jpg"
fi

# ---- Summary: constant → confirmed path (paste confirmed values into driver.go) ----
log ""
log "════════════════════════════ SUMMARY ════════════════════════════"
log "$(printf '%-26s  %s' 'CONSTANT' 'RESULT')"
log "------------------------------------------------------------------"
for row in "${RESULTS[@]}"; do
  log "$(printf '%-26s  %s' "${row%%|*}" "${row#*|}")"
done
log "=================================================================="
log ""
log "Next: for each CONFIRMED path, update the constant in source/driver.go, drop its"
log "PLACEHOLDER tag, update the mock in driver_test.go to match the real response, and"
log "run: export PATH=\"/opt/homebrew/bin:\$PATH\" && go test ./source/ -v"
log ""
log "If everything is NEEDS-PROBE: load ${BASE}/ in a browser and inspect the web UI's"
log "JavaScript for fetch/XHR calls — that reveals the real REST paths fastest. VISCA over"
log "IP (port 52381) and ONVIF (port 81) are documented, standardized fallbacks."
