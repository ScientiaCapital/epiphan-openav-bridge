# Phase 1: RTSP Compatibility Proof

Proves that Epiphan EC20 and Pearl RTSP streams ingest cleanly into ffmpeg — the same recording pipeline [OpenAV](https://github.com/Dartmouth-OpenAV) uses in production with Magewell converters.

## What This Tests

1. **Stream discovery** — find RTSP URLs for EC20 main/sub streams and Pearl channel output
2. **Stream analysis** — extract codec, resolution, framerate, bitrate via ffprobe
3. **Recording** — verify `ffmpeg -c copy` (stream copy, no re-encoding) produces clean MP4 files
4. **Comparison** — document how EC20/Pearl streams compare to Magewell output parameters

If these streams work with `ffmpeg -c copy`, they're drop-in compatible with OpenAV's existing capture pipeline.

## Prerequisites

- **Python 3.11+**
- **ffmpeg** (includes ffprobe)
  - macOS: `brew install ffmpeg`
  - Ubuntu: `sudo apt install ffmpeg`

No Python package dependencies — the script uses only the standard library.

## Usage

```bash
# Probe EC20 streams (no recording)
python rtsp_test.py --ec20-ip 192.168.1.50 --probe-only

# Probe Pearl streams (no recording)
python rtsp_test.py --pearl-ip 192.168.1.100 --probe-only

# Full test: probe + record 30s clips from both devices
python rtsp_test.py --ec20-ip 192.168.1.50 --pearl-ip 192.168.1.100

# Custom duration and JSON output
python rtsp_test.py --ec20-ip 192.168.1.50 --duration 60 --json
```

## Output

- **Console**: Stream parameters and recording results
- **MP4 files**: Test recordings in this directory (gitignored)
- **JSON** (with `--json` flag): Machine-readable results

## Results

See [stream_analysis.md](stream_analysis.md) for documented stream parameters and compatibility assessment.

## Context

OpenAV records lectures using ffmpeg ingesting RTSP streams from Magewell HDMI-to-RTSP converters. This proof verifies that Epiphan devices produce RTSP output compatible with that same pipeline, which is the prerequisite for building OpenAV microservices in Phase 2.

For more on OpenAV's architecture, see the [OpenAV wiki](https://github.com/Dartmouth-OpenAV/.github/wiki).
