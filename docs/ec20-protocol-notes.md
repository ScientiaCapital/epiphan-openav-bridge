# EC20 API Discovery Program

## Current implementation (as of 2026-11-11)

This section describes what the shipped driver in `openav-epiphan-ec20/source/` does. Where
it disagrees with anything further down, this section wins: the rest of the file is the
discovery history and is kept as written, with superseded parts marked.

**Motion, presets, position and health: raw VISCA over TCP :5678** (`visca.go`)

- `viscaSend` dials `<device host>:5678` (`viscaPort`), writes one raw VISCA frame (no Sony
  8-byte UDP header), skips ACKs (`90 4y FF`), returns the completion (`90 5y FF`) and turns
  an error reply (`90 6y FF`) into an error. Connect and each read/write are bounded by
  `viscaDialTimeout` (5 s). The port in the socketKey (the HTTP port) is ignored for VISCA,
  and so are the credentials: per the `visca.go` comments, :5678 is unauthenticated.
- `visca.go` records the transport as hardware-verified on 2026-07-18/19 (fw 3.3.40), and
  notes that Sony VISCA-over-IP on UDP :52381 (the port in Epiphan's docs) did not respond
  on that unit.
- Frames used, by driver function (`driver.go`):

  | Driver function (PUT/GET setting) | VISCA frame |
  |---|---|
  | `controlPTZ` (`ptz`) | absolute pan/tilt `81 01 06 02 VV WW p p p p t t t t FF`, then zoom direct `81 01 04 47 z z z z FF` |
  | `controlPTZHome` (`ptzhome`) | `81 01 06 04 FF` |
  | `recallPreset` (`preset`) | `81 01 04 3F 02 nn FF` |
  | `savePreset` (`presetsave`) | `81 01 04 3F 01 nn FF` (the preset name is validated but not sent) |
  | `jogPTZ` (`jog`) | pan-tilt drive `81 01 06 01 VV WW <panDir> <tiltDir> FF`; `stop` sends `03 03` |
  | `getCameraStatus` (`status`), `getPTZPosition` (`ptzposition`) | inquiries `81 09 06 12 FF` (pan/tilt) and `81 09 04 47 FF` (zoom); values are raw VISCA units |
  | `healthCheck` (`healthcheck`) | version inquiry `81 09 00 02 FF` |

- `controlPTZ` takes degrees and validates pan -162.5..162.5 and tilt -30..90 (it refuses
  NaN/Inf first). The degrees-to-units scale (`panUnitsPerDegree`, `tiltUnitsPerDegree` =
  14.0) is marked PLACEHOLDER in `driver.go` and still needs calibrating on hardware. Zoom
  is passed as a raw VISCA zoom position clamped to 0..0x4000 (`zoomMax`, marked
  NEEDS-PROBE).
- `getPresets` returns an "unsupported" string: VISCA has no list-presets inquiry.
- `getPreview` makes no device call; it returns `rtsp://<host>:554/1`.

**AI tracking only: CGI with an app-layer session token** (`cgiauth.go`)

- `controlTracking` (`tracking`) is the only feature on the CGI surface. It sends an HTTP
  **GET** (not POST) to the path from `ec20TrackingCommand`:
  `/cgi-bin/ptzctrl.cgi?post_aimode&Single_Track` (presenter, the default),
  `...&Frame_Track` (zone) or `...&Off` (disable). The code marks this command
  "CONFIRM ON HARDWARE": it is not in public docs, and `/cgi-bin/vip?set_ai_vip&vip=<1|0>`
  is the recorded alternative.
- Before the command, `ec20Login` runs the `/cgi-bin/auth.cgi` handshake (see Appendix A
  below) and the `authorization` token it returns is sent as a lowercase `authorization`
  request header. If lighttpd answers with a transport HTTP Digest challenge, `ec20CGIDo`
  answers it (MD5, qop=auth) while keeping the app token as a separate header line.
- Tokens are cached per device host **and** credentials (key: host plus SHA-256 of
  `user:password`; the plaintext password is never a key), for at most 10 minutes
  (`cgiSessionTTL`), in a cache of at most 64 entries (`cgiSessionMaxEntries`). A bare 401
  drops the cached token, triggers exactly one fresh login and one retry; a second 401
  drops that token too and returns an error.
- `param.cgi`, `login.cgi`, the `jwt` header and the WebSocket on :4567 are **not used**
  by the driver. The `jwt` value from auth.cgi is stored but never sent.

---

> ✅ **REDESIGN COMPLETE (sprint `feat/ec20-hybrid-driver`, 2026-07-21).** The driver no longer uses the
> RESTful `/api/*` placeholder model this program set out to discover — that model was wrong. It now runs
> on the real, hardware-verified planes: **VISCA-over-IP (raw, TCP :5678)** for PTZ/presets/home/position,
> and **CGI** (`/cgi-bin/auth.cgi` session token + `ptzctrl.cgi`/`param.cgi`) for AI tracking + status.
> The placeholder-probe workflow below is kept for history and for the two remaining CONFIRM-ON-HARDWARE
> items: the exact AI-tracking command and the degrees→VISCA-units PTZ calibration. See the
> "🔴 LIVE HARDWARE DISCOVERY" section for the confirmed contract.

Autoresearch-style program for systematically discovering the Epiphan EC20 PTZ camera's REST API endpoints.

Inspired by [Karpathy's autoresearch](https://github.com/karpathy/autoresearch): one modifiable file, one metric, fixed time budget, iterate autonomously.

---

## Constraints

- **One modifiable file:** `openav-epiphan-ec20/source/driver.go` (endpoint constants only, lines 23-34)
- **One metric:** HTTP response code — 200 = confirmed, 4xx = discard, 3xx = follow redirect
  > ⚠️ **Falsified on hardware 2026-07-18:** this device returns **HTTP 200 with body
  > `{"err":"Invalid API command"}`** for unknown `/api/` commands. HTTP status is NOT a
  > reliable signal — the JSON body must be inspected. See LIVE HARDWARE DISCOVERY below.
- **Fixed time budget:** 30 seconds per endpoint probe
- **Requires:** Real EC20 camera accessible on local network

## Prerequisites

```bash
export EC20_HOST=192.0.2.10    # Real EC20 IP
export EC20_USERNAME=admin       # Device credentials
export EC20_PASSWORD=changeme
```

## The 12 Unknown Endpoints

Each endpoint below has a PLACEHOLDER path and a list of candidate alternatives to probe.

### 1. Device Status
```
Current:    /api/status
Candidates: /api/v1/status, /status, /api/device/status, /api/info,
            /api/v1/info, /api/system/status, /cgi-bin/status
Metric:     GET → 200 with JSON body containing device info
```

### 2. PTZ Position
```
Current:    /api/ptz/position
Candidates: /api/v1/ptz/position, /ptz/position, /api/ptz/query,
            /cgi-bin/ptz.cgi?action=getStatus, /api/ptz/status,
            /ISAPI/PTZCtrl/channels/1/status
Metric:     GET → 200 with JSON containing pan/tilt/zoom values
```

### 3. PTZ Pan
```
Current:    /api/ptz/pan
Candidates: /api/v1/ptz/pan, /api/ptz/move, /api/ptz/continuous,
            /cgi-bin/ptz.cgi?action=pan, /ISAPI/PTZCtrl/channels/1/continuous
Metric:     POST {degrees, speed} → 200
```

### 4. PTZ Tilt
```
Current:    /api/ptz/tilt
Candidates: /api/v1/ptz/tilt, /api/ptz/move, /api/ptz/continuous,
            /cgi-bin/ptz.cgi?action=tilt
Metric:     POST {degrees, speed} → 200
```

### 5. PTZ Zoom
```
Current:    /api/ptz/zoom
Candidates: /api/v1/ptz/zoom, /api/ptz/zoom/set, /api/zoom,
            /cgi-bin/ptz.cgi?action=zoom
Metric:     POST {level} → 200
```

### 6. PTZ Home
```
Current:    /api/ptz/home
Candidates: /api/v1/ptz/home, /api/ptz/preset/home, /api/ptz/goto/home,
            /cgi-bin/ptz.cgi?action=home
Metric:     POST (no body) → 200
```

### 7. Preset List
```
Current:    /api/ptz/presets
Candidates: /api/v1/ptz/presets, /api/presets, /api/ptz/preset/list,
            /cgi-bin/ptz.cgi?action=getPresets
Metric:     GET → 200 with JSON array of presets
```

### 8. Preset Goto
```
Current:    /api/ptz/preset/goto
Candidates: /api/v1/ptz/preset/goto, /api/ptz/preset/call,
            /api/ptz/goto, /cgi-bin/ptz.cgi?action=gotoPreset
Metric:     POST {preset_id} → 200
```

### 9. Preset Save
```
Current:    /api/ptz/preset/save
Candidates: /api/v1/ptz/preset/save, /api/ptz/preset/set,
            /api/ptz/preset/store, /cgi-bin/ptz.cgi?action=setPreset
Metric:     POST {preset_id, name} → 200
```

### 10. Tracking Enable
```
Current:    /api/tracking/enable
Candidates: /api/v1/tracking/enable, /api/tracking/start,
            /api/tracking/on, /api/ai/tracking/enable
Metric:     POST {mode} → 200
```

### 11. Tracking Disable
```
Current:    /api/tracking/disable
Candidates: /api/v1/tracking/disable, /api/tracking/stop,
            /api/tracking/off, /api/ai/tracking/disable
Metric:     POST (no body) → 200
```

### 12. Preview Image
```
Current:    /api/preview
Candidates: /api/v1/preview, /api/snapshot, /api/image,
            /cgi-bin/snapshot.cgi, /ISAPI/Streaming/channels/1/picture,
            /api/capture, /preview.jpg
Metric:     GET → 200 with Content-Type: image/jpeg
```

## Discovery Protocol

For each endpoint:

1. **Probe current placeholder** — maybe we guessed right
2. **Try candidates** in order — first 200 wins
3. **Log result** to discovery log below
4. **If all candidates fail:** try common API discovery:
   - `GET /api/` — may return endpoint list
   - `GET /api/v1/` — versioned variant
   - `OPTIONS /` — may reveal routes
   - Check for Swagger/OpenAPI at `/api/docs`, `/swagger.json`, `/openapi.json`

## Doc Research — 2026-07-17 (no hardware)

Mined Epiphan's own documentation to de-risk the endpoints without a device. The
**REST URL paths remain undiscoverable from public docs** (neither the AI User Guide nor
the Q-SYS Plugin README lists a single `/api/...` path; the User Guide PDF is image-only),
so the 12 paths below stay `NEEDS-PROBE`. But several **behavioral facts are now
DOC-CONFIRMED** and were applied directly to `driver.go` — no hardware required.

**Sources**
- Epiphan Knowledge `ec20/epiphan-ec20-ai-userguide` (Epiphan-EC20-AI-UserGuide.docx, updated 2026-07-16) — authoritative
- Epiphan Knowledge `ec20/epiphan-ec20-qsys-plugin-readme` (2026-05-12)
- EC20 tech-specs / user-guide portal (epiphan.com)

**DOC-CONFIRMED (applied to driver.go)**
| Fact | Value | Source |
|------|-------|--------|
| Preset number range | **0–255** — preset **0 is valid** ("If preset 0 is saved, PTZ will be moved to preset 0"); "Presets max: 255" | AI User Guide specs + init note |
| AI tracking modes | **`Presenter`** (default, aka Human Tracking) and **`Zone`** | AI User Guide, Tracking Configuration |
| Pan range | **±162.5°** | AI User Guide, PTZ specs |
| Tilt range | **−30° → +90°** | AI User Guide, PTZ specs |
| Pan speed | 1.8°–80°/s | AI User Guide, PTZ specs |
| Tilt speed | 1.5°–49°/s | AI User Guide, PTZ specs |
| HTTP port | **80** (default; configurable 1025–65535) | AI User Guide, Network > Port |
| Auth | ~~HTTP Basic~~ **HTTP Digest** (MD5/qop=auth), default `admin` / `admin` | ⚠️ corrected on hardware 2026-07-18 — see LIVE DISCOVERY; Basic → 401 always |
| Preview stream | MJPEG (preview stream is second stream) | AI User Guide + Q-SYS README |

> ⚠️ The Q-SYS plugin README shows presets "0–11" — that is only the *plugin's* default
> preset **count** (a configurable property), NOT the camera's range. The camera range is
> 0–255 per the AI User Guide specs. Trust the User Guide.

**Alternative documented control planes (NEEDS-PROBE / future work)**
- **VISCA over IP** — port **52381** (fixed, cannot be changed). Standardized binary protocol; a robust fallback if the proprietary REST paths never surface.
- **ONVIF** — port **81** by default (toggle + auth in Network > ONVIF). Standardized PTZ profile.
- **NDI|HX3** control, RS-232/RS-485 (VISCA/Pelco-D/Pelco-P).

**Best hardware-day shortcut (per "First Move" below):** the EC20 has a full web UI on
port 80 — inspect its JavaScript for the real `fetch`/XHR calls. That will reveal the REST
path structure faster than brute-forcing the candidate lists.

---

## 🔴 LIVE HARDWARE DISCOVERY — 2026-07-18 (device @ 192.0.2.11, its admin credentials)

First live session against a real EC20. **Two headline findings supersede the assumptions above.**
Full session fingerprint captured in scratch `ec20_live_findings.md`.

### 1. Auth is HTTP **Digest**, not Basic — **FIXED in driver.go**
`lighttpd/1.4.75` answers every request with `WWW-Authenticate: Digest realm="", algorithm=MD5,
qop="auth"`. Basic → **401** always; `curl --digest -u admin:<password> → **200**. The driver's
`req.SetBasicAuth` was broken against real hardware regardless of credentials.
- **Fix landed:** `ec20APIRequest` → `ec20DoWithDigest`, an RFC 2617 MD5/qop=auth handshake using
  stdlib only (`crypto/md5`, `crypto/rand`) — **no new dependency**. Falls back to Basic for
  non-Digest servers (keeps mocks green). `ec20_probe.sh` now uses `curl --digest`.
- **Tests:** `TestEC20APIGet_DigestAuth_Success` / `_WrongPassword` with a mock that recomputes and
  validates the digest response (not a stub). Full suite green.

### 2. The real API is **CGI/VISCA**, not RESTful `/api/*` — all 12 placeholders are wrong in *model*
Reverse-engineered from the web-UI JS (`/js/build-new.min.js`). The device is a rebadged Chinese ODM
AI-tracking PTZ camera; the `ptzctrl.cgi`/`param.cgi` grammar matches **SMTAV** BA20S/BA30S (publicly
documented). Epiphan's *official* integration story is VISCA-over-IP :52381 + ONVIF — they do **not**
publicly document this CGI surface (no Swagger/REST spec exists, unlike Pearl).

> **Superseded (see "Current implementation" at the top).** The driver does not route PTZ,
> zoom, home or presets through `ptzctrl.cgi`; those use raw VISCA over TCP :5678. Of this
> table only the AI-tracking row (`post_aimode`) is used, sent as a GET. Kept as discovery
> history.

- **PTZ / zoom / focus / preset / tracking** → `POST /cgi-bin/ptzctrl.cgi?ptzcmd&...`
  (public grammar, cross-checked vs SMTAV):

  | Action | Request |
  |--------|---------|
  | Pan/tilt | `ptzctrl.cgi?ptzcmd&<up\|down\|left\|right\|leftup\|rightup\|leftdown\|rightdown\|ptzstop>&<panSpeed 1-24>&<tiltSpeed 1-20>` |
  | Zoom | `ptzctrl.cgi?ptzcmd&<zoomin\|zoomout\|zoomstop>&<speed 1-7>` |
  | Focus | `ptzctrl.cgi?ptzcmd&<focusin\|focusout\|focusstop>&<speed 1-7>` |
  | Preset save/call | `ptzctrl.cgi?ptzcmd&<posset\|poscall>&<N>` (0-89 range1, 100-254 range2) |
  | Home | `ptzctrl.cgi?ptzcmd&home` |
  | Absolute PTZ | `ptzctrl.cgi?ptzcmd&ABS&<panSpd>&<tiltSpd>&<panPos hex4>&<tiltPos hex4>` |
  | Absolute zoom | `ptzctrl.cgi?ptzcmd&zoomto&<speed>&<pos 0000-4000>` |
  | AI tracking | `ptzctrl.cgi?post_aimode&<Single_Track\|Frame_Track\|Demo_Mode\|Off>`; status `?get_aimode` → VISCA `90 50 0p FF` (02=On, 03=Off) |
  | Nav/OSD | `ptzctrl.cgi?navigate_mode&<OSD\|PTZ\|CONFIRM\|OSD_BACK>`, `?osdcmd`, `?post_image_value` |

- **Config / status / record / stream / tracking-status** → `/cgi-bin/param.cgi?<command>` (~70
  commands; **no public docs** — live fingerprint is the only source): `get_device_conf`,
  `get_system_conf`, `get_target_status`(+`_head_face`)=tracking, `get_record_info`,
  `udisk_save_record`, `post_streaming`, `get_media_video`/`post_media_video`, `post_reboot`,
  `post_visca`, `get_ndi_info`, `get_roi`/`post_roi_conf`, `save_profile`/`apply_profile`/`remove_profile`.
- **App-layer session (above Digest)**: `/cgi-bin/login.cgi?login|?logout`, `/cgi-bin/auth.cgi`; UI
  sends a `jwt` request header; password MD5-hashed client-side. `param.cgi` GETs **HANG** with
  Digest alone → they need the JWT session too. **No public docs.**
- **Realtime**: WebSocket `ws://<host>:4567/ws/`. **No public docs.**
- Quirk: unknown `/api/` commands return **200** with `{"err":"Invalid API command"}`.

### Redesign scope (DEFERRED — separate session)

> **Superseded.** The redesign shipped differently from this plan: VISCA over TCP :5678 for
> motion and status, and `auth.cgi` (not `login.cgi`/JWT) for the tracking command only.
> See "Current implementation" at the top.

A correct EC20 driver must: (a) HTTP Digest [**DONE**], (b) `login.cgi` JWT session, (c) `param.cgi` +
`ptzctrl.cgi?ptzcmd` VISCA-style command model. This is a driver **redesign**, not a constant swap.
PTZ/preset/zoom/tracking grammar is public (SMTAV); param.cgi/JWT/WebSocket are not.

**Public ground-truth sources for the PTZ layer:**
- SMTAV HTTP-CGI control — https://www.smtav.com/blogs/how-to-use-the-camera/http-cgi-control
- SMTAV AI-tracking — https://www.smtav.com/blogs/how-to-use-the-camera/how-to-use-ai-tracking-cameras-tracking-function
- SMTAV BA20S / BA30S manuals (ManualsLib)

---

## Discovery Log

> **Superseded (see "Current implementation" at the top).** The "Real target (CGI)" column
> was the plan for the redesign, not what shipped: Status, Position, Pan/Tilt, Zoom, Home,
> Preset Goto and Preset Save are VISCA over TCP :5678; Presets is unsupported; Preview is
> an RTSP URL; only Tracking On/Off use `ptzctrl.cgi?post_aimode`. Kept as discovery
> history.

**All 12 RESTful placeholders are wrong in MODEL** (real API is CGI — see LIVE DISCOVERY above).
Superseded 2026-07-18. Real targets below are ground-truth mappings for the deferred redesign.

| Endpoint | Old placeholder (WRONG) | Real target (CGI) | Source |
|----------|-------------------------|-------------------|--------|
| Status | /api/status | `param.cgi?get_device_conf` / `get_system_conf` | live fingerprint |
| Position | /api/ptz/position | (query via VISCA/`ptzctrl.cgi` inquiry — TBD) | live |
| Pan | /api/ptz/pan | `ptzctrl.cgi?ptzcmd&<left\|right\|...>&<spd>` | SMTAV + live |
| Tilt | /api/ptz/tilt | `ptzctrl.cgi?ptzcmd&<up\|down\|...>&<spd>` | SMTAV + live |
| Zoom | /api/ptz/zoom | `ptzctrl.cgi?ptzcmd&<zoomin\|zoomout\|zoomstop>&<spd>` / `zoomto` | SMTAV + live |
| Home | /api/ptz/home | `ptzctrl.cgi?ptzcmd&home` | SMTAV + live |
| Presets | /api/ptz/presets | (list via `param.cgi` — TBD) | live |
| Preset Goto | /api/ptz/preset/goto | `ptzctrl.cgi?ptzcmd&poscall&<N>` | SMTAV + live |
| Preset Save | /api/ptz/preset/save | `ptzctrl.cgi?ptzcmd&posset&<N>` | SMTAV + live |
| Tracking On | /api/tracking/enable | `ptzctrl.cgi?post_aimode&Single_Track` | SMTAV + live |
| Tracking Off | /api/tracking/disable | `ptzctrl.cgi?post_aimode&Off` | SMTAV + live |
| Preview | /api/preview | MJPEG 2nd stream (RTSP/HTTP — TBD) + `param.cgi?get_media_video` | live |
| **Auth** | ~~Basic~~ | **HTTP Digest MD5/qop=auth — DONE** + `login.cgi` JWT session (TBD) | live |

## After Discovery

When an endpoint is confirmed:

1. Update the constant in `openav-epiphan-ec20/source/driver.go` (remove PLACEHOLDER comment)
2. Update the mock server in `driver_test.go` to match the real response format
3. Run `go test ./source/ -v` to verify tests still pass
4. Log confirmation in this discovery log with date

## First Move

Before probing individual endpoints, try API discovery:

```bash
# Does the EC20 expose an API index?
curl -u $EC20_USERNAME:$EC20_PASSWORD http://$EC20_HOST/api/ -v
curl -u $EC20_USERNAME:$EC20_PASSWORD http://$EC20_HOST/api/v1/ -v
curl -u $EC20_USERNAME:$EC20_PASSWORD http://$EC20_HOST/swagger.json -v
curl -u $EC20_USERNAME:$EC20_PASSWORD http://$EC20_HOST/ -v

# Check the web UI — often reveals API patterns in JavaScript
curl -u $EC20_USERNAME:$EC20_PASSWORD http://$EC20_HOST/ -s | grep -i "api\|fetch\|ajax\|endpoint"
```

If the web UI loads, inspect its JavaScript for API calls — this often reveals the real endpoint structure faster than brute-force probing.

---

## Appendix A — auth.cgi handshake, as implemented

Referenced from `cgiauth.go`. Summarised from that file's comments and code
(`parseCGIChallenge`, `ec20CGIResponse`, `ec20CGIAuthHeader`, `ec20Login`), which were
reverse-engineered from the device web UI (`build-new.min.js`). All requests are
`GET /cgi-bin/auth.cgi`.

1. Without an `authenticate` header the device answers 401 with a custom challenge:
   `WWW-Authenticate: base64(realm) "." base64(nonce) "." base64(qop)`.
2. `HA1 = md5(user:realm:pass)`, `HA2 = md5("GET:/auth.cgi")` (the short URI, although
   the request path is `/cgi-bin/auth.cgi`),
   `response = md5(HA1:nonce:nc:cnonce:HA2)`. `nc` is a raw integer (the driver sends 1,
   not zero-padded) and qop is not part of the hash. The cnonce is 32 hex characters.
3. Resend with the request header
   `authenticate: username="..",nonce="..",nc=1,cnonce="..",uri="/auth.cgi",response=".."`.
4. HTTP 200 returns `jwt` and `authorization` response headers. Per the code comments,
   201 means bad credentials and 500 a server error; the driver treats any non-200 as a
   failed login.
