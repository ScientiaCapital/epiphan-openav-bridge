#!/usr/bin/env python3
"""
RTSP Compatibility Proof for epiphan-openav-bridge

Tests that Epiphan EC20 and Pearl RTSP streams ingest cleanly into ffmpeg,
matching the pipeline OpenAV uses at Dartmouth with Magewell converters.

Usage:
    python rtsp_test.py --ec20-ip 192.168.1.50
    python rtsp_test.py --pearl-ip 192.168.1.100
    python rtsp_test.py --ec20-ip 192.168.1.50 --pearl-ip 192.168.1.100
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Optional


@dataclass
class StreamInfo:
    """Parsed stream parameters from ffprobe."""
    device: str
    url: str
    codec: Optional[str] = None
    width: Optional[int] = None
    height: Optional[int] = None
    framerate: Optional[str] = None
    bitrate: Optional[str] = None
    pixel_format: Optional[str] = None
    profile: Optional[str] = None
    level: Optional[str] = None
    error: Optional[str] = None


def check_ffmpeg() -> bool:
    """Verify ffmpeg and ffprobe are installed."""
    for tool in ("ffmpeg", "ffprobe"):
        try:
            subprocess.run(
                [tool, "-version"],
                capture_output=True,
                check=True,
            )
        except FileNotFoundError:
            print(f"Error: {tool} not found. Install ffmpeg first.")
            print("  macOS:   brew install ffmpeg")
            print("  Ubuntu:  sudo apt install ffmpeg")
            return False
    return True


def probe_stream(device_name: str, rtsp_url: str, timeout: int = 10) -> StreamInfo:
    """
    Run ffprobe on an RTSP stream and extract parameters.

    Args:
        device_name: Human-readable device identifier (e.g., "EC20", "Pearl")
        rtsp_url: Full RTSP URL to probe
        timeout: Seconds to wait before giving up

    Returns:
        StreamInfo with parsed parameters or error message
    """
    info = StreamInfo(device=device_name, url=rtsp_url)

    try:
        result = subprocess.run(
            [
                "ffprobe",
                "-v", "quiet",              # Suppress ffprobe banner
                "-print_format", "json",    # Machine-readable output
                "-show_streams",            # Stream-level detail
                "-rtsp_transport", "tcp",   # TCP transport (more reliable than UDP)
                rtsp_url,
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        info.error = f"Timed out after {timeout}s — device may be unreachable"
        return info
    except Exception as e:
        info.error = str(e)
        return info

    if result.returncode != 0:
        stderr = result.stderr.strip()
        info.error = stderr if stderr else f"ffprobe exited with code {result.returncode}"
        return info

    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        info.error = "Failed to parse ffprobe JSON output"
        return info

    # Find the video stream (there may also be audio streams)
    streams = data.get("streams", [])
    video = next((s for s in streams if s.get("codec_type") == "video"), None)

    if not video:
        info.error = "No video stream found"
        return info

    # codec_name: "h264", "hevc", etc.
    info.codec = video.get("codec_name")

    # Resolution
    info.width = video.get("width")
    info.height = video.get("height")

    # Framerate: r_frame_rate is the "real" framerate (e.g., "30/1")
    info.framerate = video.get("r_frame_rate")

    # Bitrate: may be in stream or need to be inferred from container
    info.bitrate = video.get("bit_rate")

    # pixel format: "yuv420p", "nv12", etc. — determines decoder compatibility
    info.pixel_format = video.get("pix_fmt")

    # H.264/H.265 profile and level — determines decoder requirements
    info.profile = video.get("profile")
    info.level = str(video.get("level")) if video.get("level") is not None else None

    return info


def record_clip(
    device_name: str,
    rtsp_url: str,
    output_dir: Path,
    duration: int = 30,
    timeout: int = 45,
) -> Optional[Path]:
    """
    Record a test clip from an RTSP stream using ffmpeg.

    Uses -c copy (stream copy) to avoid re-encoding — this matches how
    OpenAV's ffmpeg pipeline ingests RTSP streams.

    Args:
        device_name: Used for output filename
        rtsp_url: Full RTSP URL to record
        output_dir: Directory for output file
        duration: Recording length in seconds
        timeout: Subprocess timeout (should exceed duration)

    Returns:
        Path to recorded file, or None on failure
    """
    output_file = output_dir / f"{device_name.lower()}_test_{duration}s.mp4"

    print(f"  Recording {duration}s clip from {device_name}...")
    print(f"  Output: {output_file}")

    try:
        result = subprocess.run(
            [
                "ffmpeg",
                "-y",                       # Overwrite output file
                "-rtsp_transport", "tcp",   # TCP transport
                "-i", rtsp_url,             # Input RTSP stream
                "-c", "copy",               # Stream copy (no re-encoding)
                "-t", str(duration),         # Duration in seconds
                str(output_file),
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        print(f"  Error: Recording timed out after {timeout}s")
        return None
    except Exception as e:
        print(f"  Error: {e}")
        return None

    if result.returncode != 0:
        stderr_lines = result.stderr.strip().split("\n")
        # ffmpeg writes progress to stderr; only show last few lines for errors
        error_lines = [l for l in stderr_lines if "error" in l.lower() or "fail" in l.lower()]
        if error_lines:
            print(f"  Error: {error_lines[-1]}")
        else:
            print(f"  Error: ffmpeg exited with code {result.returncode}")
        return None

    if output_file.exists() and output_file.stat().st_size > 0:
        size_mb = output_file.stat().st_size / (1024 * 1024)
        print(f"  Success: {size_mb:.1f} MB recorded")
        return output_file

    print("  Error: Output file is empty or missing")
    return None


def build_rtsp_urls(ip: str, device_type: str) -> list[str]:
    """
    Build candidate RTSP URLs for a device.

    EC20 and Pearl use different URL patterns. We try common patterns
    and let ffprobe determine which one responds.
    """
    if device_type == "ec20":
        return [
            f"rtsp://{ip}:554/stream1",     # Main stream (typically 4K or 1080p)
            f"rtsp://{ip}:554/stream2",     # Sub stream (typically 720p or lower)
        ]
    elif device_type == "pearl":
        return [
            f"rtsp://{ip}:554/stream.sdp",  # Default channel output
            f"rtsp://{ip}:554/stream1.sdp",  # Channel 1 explicit
        ]
    return []


def print_stream_info(info: StreamInfo) -> None:
    """Print stream analysis results in a readable format."""
    print(f"\n{'=' * 60}")
    print(f"  {info.device}")
    print(f"  {info.url}")
    print(f"{'=' * 60}")

    if info.error:
        print(f"  ERROR: {info.error}")
        return

    print(f"  Codec:        {info.codec}")
    print(f"  Resolution:   {info.width}x{info.height}")
    print(f"  Framerate:    {info.framerate}")
    print(f"  Pixel format: {info.pixel_format}")
    print(f"  Profile:      {info.profile}")
    print(f"  Level:        {info.level}")
    if info.bitrate:
        bitrate_mbps = int(info.bitrate) / 1_000_000
        print(f"  Bitrate:      {bitrate_mbps:.1f} Mbps")
    else:
        print(f"  Bitrate:      (not reported — measure from recording)")


def run_tests(
    ec20_ip: Optional[str],
    pearl_ip: Optional[str],
    duration: int,
    output_dir: Path,
    probe_only: bool,
) -> dict:
    """
    Run RTSP discovery, analysis, and recording tests.

    Returns a results dict suitable for JSON export.
    """
    results: dict = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "streams": [],
        "recordings": [],
    }

    # --- EC20 ---
    if ec20_ip:
        print("\n--- EC20 RTSP Stream Analysis ---")
        for url in build_rtsp_urls(ec20_ip, "ec20"):
            info = probe_stream("EC20", url)
            print_stream_info(info)
            results["streams"].append(asdict(info))

            # Record from the first working stream
            if not probe_only and info.error is None:
                clip = record_clip("EC20", url, output_dir, duration)
                if clip:
                    results["recordings"].append({
                        "device": "EC20",
                        "url": url,
                        "file": str(clip),
                        "duration": duration,
                    })

    # --- Pearl ---
    if pearl_ip:
        print("\n--- Pearl RTSP Stream Analysis ---")
        for url in build_rtsp_urls(pearl_ip, "pearl"):
            info = probe_stream("Pearl", url)
            print_stream_info(info)
            results["streams"].append(asdict(info))

            if not probe_only and info.error is None:
                clip = record_clip("Pearl", url, output_dir, duration)
                if clip:
                    results["recordings"].append({
                        "device": "Pearl",
                        "url": url,
                        "file": str(clip),
                        "duration": duration,
                    })

    return results


def main() -> int:
    parser = argparse.ArgumentParser(
        description="RTSP compatibility proof for Epiphan EC20 and Pearl with ffmpeg",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --ec20-ip 192.168.1.50
  %(prog)s --pearl-ip 192.168.1.100
  %(prog)s --ec20-ip 192.168.1.50 --pearl-ip 192.168.1.100
  %(prog)s --ec20-ip 192.168.1.50 --probe-only
        """,
    )

    parser.add_argument(
        "--ec20-ip",
        help="EC20 PTZ camera IP address",
    )
    parser.add_argument(
        "--pearl-ip",
        help="Pearl encoder IP address",
    )
    parser.add_argument(
        "--duration",
        type=int,
        default=30,
        help="Recording duration in seconds (default: 30)",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path(__file__).parent,
        help="Directory for output files (default: proof/)",
    )
    parser.add_argument(
        "--probe-only",
        action="store_true",
        help="Only probe streams, don't record clips",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON",
    )

    args = parser.parse_args()

    if not args.ec20_ip and not args.pearl_ip:
        parser.error("At least one of --ec20-ip or --pearl-ip is required")

    if not check_ffmpeg():
        return 1

    args.output_dir.mkdir(parents=True, exist_ok=True)

    results = run_tests(
        ec20_ip=args.ec20_ip,
        pearl_ip=args.pearl_ip,
        duration=args.duration,
        output_dir=args.output_dir,
        probe_only=args.probe_only,
    )

    if args.json:
        print("\n" + json.dumps(results, indent=2))

    # Summary
    working = [s for s in results["streams"] if s.get("error") is None]
    failed = [s for s in results["streams"] if s.get("error") is not None]

    print(f"\n{'=' * 60}")
    print(f"  Summary: {len(working)} streams detected, {len(failed)} failed")
    if results["recordings"]:
        print(f"  Recordings: {len(results['recordings'])} clips saved")
    print(f"{'=' * 60}")

    return 0 if working else 1


if __name__ == "__main__":
    sys.exit(main())
