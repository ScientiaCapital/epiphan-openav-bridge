#!/usr/bin/env bash
# generate-config.sh — Generates smart-room-demo.json from environment variables.
# Usage: ./generate-config.sh [path-to-env-file]
#   Default env file: .env in the same directory as this script.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${1:-$SCRIPT_DIR/.env}"
OUTPUT_FILE="$SCRIPT_DIR/system-configs/smart-room-demo.json"

# Read environment file without shell expansion
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: env file not found: $ENV_FILE" >&2
    exit 1
fi

# Parse env vars safely (no shell expansion of values)
PEARL_HOST="" PEARL_USERNAME="" PEARL_PASSWORD=""
EC20_HOST="" EC20_USERNAME="" EC20_PASSWORD=""

while IFS= read -r line || [ -n "$line" ]; do
    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// /}" ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    case "$key" in
        PEARL_HOST)     PEARL_HOST="$val" ;;
        PEARL_USERNAME) PEARL_USERNAME="$val" ;;
        PEARL_PASSWORD) PEARL_PASSWORD="$val" ;;
        EC20_HOST)      EC20_HOST="$val" ;;
        EC20_USERNAME)  EC20_USERNAME="$val" ;;
        EC20_PASSWORD)  EC20_PASSWORD="$val" ;;
    esac
done < "$ENV_FILE"

# Validate required variables
MISSING=""
[ -z "$PEARL_HOST" ]     && MISSING="$MISSING PEARL_HOST"
[ -z "$PEARL_USERNAME" ] && MISSING="$MISSING PEARL_USERNAME"
[ -z "$PEARL_PASSWORD" ] && MISSING="$MISSING PEARL_PASSWORD"
[ -z "$EC20_HOST" ]      && MISSING="$MISSING EC20_HOST"
[ -z "$EC20_USERNAME" ]  && MISSING="$MISSING EC20_USERNAME"
[ -z "$EC20_PASSWORD" ]  && MISSING="$MISSING EC20_PASSWORD"

if [ -n "$MISSING" ]; then
    echo "Error: missing required environment variables:$MISSING" >&2
    exit 1
fi

# Generate config JSON from embedded template (quoted heredoc — no expansion)
cat > "$OUTPUT_FILE" <<'ENDJSON'
{
  "system_name": "Smart Room Demo",
  "control_sets": {
    "recording": {
      "name": "Recording",
      "icon": "screen",
      "controls": {
        "record": {
          "type": "power",
          "channel": "main",
          "value": {
            "set": [
              {
                "driver": "dartmouth-openav/microservice-epiphan-pearl:current/__PEARL_USERNAME__:__PEARL_PASSWORD__@__PEARL_HOST__/recording",
                "method": "PUT",
                "body": "\"$on_or_off\"",
                "headers": [
                  "content-type: application/json"
                ]
              }
            ],
            "set_process": {
              "true": {
                "on_or_off": "start"
              },
              "false": {
                "on_or_off": "stop"
              }
            },
            "get": [
              "dartmouth-openav/microservice-epiphan-pearl:current/__PEARL_USERNAME__:__PEARL_PASSWORD__@__PEARL_HOST__/recordingstatus"
            ],
            "get_process": [
              "recording"
            ]
          }
        },
        "streaming": {
          "type": "power",
          "channel": "main",
          "value": {
            "set": [
              {
                "driver": "dartmouth-openav/microservice-epiphan-pearl:current/__PEARL_USERNAME__:__PEARL_PASSWORD__@__PEARL_HOST__/streaming/1",
                "method": "PUT",
                "body": "\"$on_or_off\"",
                "headers": [
                  "content-type: application/json"
                ]
              }
            ],
            "set_process": {
              "true": {
                "on_or_off": "start"
              },
              "false": {
                "on_or_off": "stop"
              }
            }
          }
        }
      }
    },
    "camera": {
      "name": "Camera",
      "icon": "camera",
      "controls": {
        "tracking": {
          "type": "power",
          "channel": "main",
          "value": {
            "set": [
              {
                "driver": "dartmouth-openav/microservice-epiphan-ec20:current/__EC20_USERNAME__:__EC20_PASSWORD__@__EC20_HOST__/tracking",
                "method": "PUT",
                "body": "\"$on_or_off\"",
                "headers": [
                  "content-type: application/json"
                ]
              }
            ],
            "set_process": {
              "true": {
                "on_or_off": "enable"
              },
              "false": {
                "on_or_off": "disable"
              }
            },
            "get": [
              "dartmouth-openav/microservice-epiphan-ec20:current/__EC20_USERNAME__:__EC20_PASSWORD__@__EC20_HOST__/status"
            ],
            "get_process": [
              "tracking"
            ]
          }
        },
        "ptz_home": {
          "type": "button",
          "channel": "main",
          "value": {
            "set": [
              {
                "driver": "dartmouth-openav/microservice-epiphan-ec20:current/__EC20_USERNAME__:__EC20_PASSWORD__@__EC20_HOST__/ptzhome",
                "method": "PUT",
                "body": "\"\"",
                "headers": [
                  "content-type: application/json"
                ]
              }
            ]
          }
        }
      }
    }
  }
}
ENDJSON

# Escape sed replacement-string special characters (\ & |)
sed_escape() {
    printf '%s' "$1" | sed -e 's/[\\&|]/\\&/g'
}

# Substitute placeholders with actual values using sed
# Use | as delimiter to avoid conflicts with / in values
sed -i '' "s|__PEARL_HOST__|$(sed_escape "$PEARL_HOST")|g" "$OUTPUT_FILE"
sed -i '' "s|__PEARL_USERNAME__|$(sed_escape "$PEARL_USERNAME")|g" "$OUTPUT_FILE"
sed -i '' "s|__PEARL_PASSWORD__|$(sed_escape "$PEARL_PASSWORD")|g" "$OUTPUT_FILE"
sed -i '' "s|__EC20_HOST__|$(sed_escape "$EC20_HOST")|g" "$OUTPUT_FILE"
sed -i '' "s|__EC20_USERNAME__|$(sed_escape "$EC20_USERNAME")|g" "$OUTPUT_FILE"
sed -i '' "s|__EC20_PASSWORD__|$(sed_escape "$EC20_PASSWORD")|g" "$OUTPUT_FILE"

echo "Generated: $OUTPUT_FILE"
