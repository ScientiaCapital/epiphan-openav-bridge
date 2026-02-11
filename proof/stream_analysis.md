# Stream Analysis Results

## EC20 PTZ Camera

### Main Stream (stream1)

| Parameter | Value |
|-----------|-------|
| RTSP URL | `rtsp://<ec20-ip>:554/stream1` |
| Codec | _pending_ |
| Resolution | _pending_ |
| Framerate | _pending_ |
| Bitrate | _pending_ |
| Pixel Format | _pending_ |
| Profile | _pending_ |
| Level | _pending_ |
| Latency (first frame) | _pending_ |

### Sub Stream (stream2)

| Parameter | Value |
|-----------|-------|
| RTSP URL | `rtsp://<ec20-ip>:554/stream2` |
| Codec | _pending_ |
| Resolution | _pending_ |
| Framerate | _pending_ |
| Bitrate | _pending_ |
| Pixel Format | _pending_ |
| Profile | _pending_ |
| Level | _pending_ |

---

## Pearl Mini

### Channel Output

| Parameter | Value |
|-----------|-------|
| RTSP URL | `rtsp://<pearl-ip>:554/stream.sdp` |
| Codec | _pending_ |
| Resolution | _pending_ |
| Framerate | _pending_ |
| Bitrate | _pending_ |
| Pixel Format | _pending_ |
| Profile | _pending_ |
| Level | _pending_ |

---

## Comparison: EC20/Pearl vs. Magewell HDMI-to-RTSP

| Parameter | EC20 (direct) | Pearl (channel) | Magewell USB Capture |
|-----------|--------------|-----------------|---------------------|
| Codec | _pending_ | _pending_ | H.264 (typical) |
| Max Resolution | _pending_ | _pending_ | 1080p60 |
| RTSP URL Pattern | `/stream1` | `/stream.sdp` | `/video` (varies) |
| Authentication | None (RTSP) | None (RTSP) | None |
| Transport | TCP/UDP | TCP/UDP | TCP/UDP |
| Additional Streams | Sub stream | Per-channel | Single |

---

## ffmpeg Pipeline Compatibility

OpenAV's recording pipeline runs ffmpeg with stream copy (`-c copy`) from RTSP sources.
Key compatibility checks:

| Check | EC20 | Pearl | Notes |
|-------|------|-------|-------|
| `ffmpeg -c copy` works | _pending_ | _pending_ | No re-encoding needed |
| Clean start (no artifacts) | _pending_ | _pending_ | First frames valid |
| Clean stop (file closes) | _pending_ | _pending_ | MP4 moov atom written |
| Simultaneous recording | _pending_ | _pending_ | Both streams at once |
| Long-duration stability | _pending_ | _pending_ | 60+ minute test |

---

## Notes

_Fill in observations, configuration changes, or issues encountered during testing._
