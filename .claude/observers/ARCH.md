# Observer: Architecture Report

**Date:** 2026-07-21 (sprint review appended; baseline below from 2026-03-21)
**Project:** epiphan-openav-bridge
**Observer Model:** observer-full (DA)
**Session:** `feat/ec20-hybrid-driver` merge review

---

## Sprint architecture review — EC20 hybrid driver (2026-07-21)

The EC20 driver moved from a single unverified REST layer to a **hybrid two-plane** model, which is
the right call and matches the real device:

- **VISCA over TCP :5678** (visca.go) — pan/tilt/zoom/home/preset/jog + position inquiries.
  Framing is hardware-verified (fw 3.3.40) and cleanly separated: pure frame builders + nibble
  codecs + one transport func, each unit-tested against a real TCP fake. Good.
- **CGI auth.cgi session** (cgiauth.go) — AI tracking only. Correctly isolated from VISCA.

### Blockers
_No architectural blockers._ The dead REST placeholder layer (the "single largest technical risk"
from the 2026-03-21 baseline, RISK on the 12 placeholder endpoint paths) is now **removed** — that
risk is retired. The `ec20APIPostJSON` / `ec20APIGetRaw` helpers dispositioned in the baseline DA
table below are gone with it.

### Risks (address before/with hardware bring-up)
[RISK] **CGI app-auth layering is unproven end-to-end.** Three coupled assumptions ride the one
untested `ec20CGIDo` transport-Digest branch: (a) app token goes in a header Go will title-case,
(b) `jwt` is not required on data calls, (c) the app token survives a co-occurring Digest challenge
(it does not — header collision, see QUALITY.md). If the real EC20 Digest-guards /cgi-bin/, tracking
breaks. Treat the CGI plane as CONFIRM-ON-HARDWARE, not done.

[RISK] **Degree↔VISCA-unit calibration is a placeholder in the shipped path.** Absolute PTZ speaks
degrees at the contract boundary but converts with an unmeasured scale, returning success on wrong
moves. Architecturally the constants are well-isolated (one-line tune), but the *default* is
uncalibrated-but-succeeds. Consider making calibration explicit state rather than a silent constant.

[RISK] **Non-atomic multi-frame commands.** `controlPTZ` opens two TCP connections (pan/tilt, then
zoom); jog/preset each open their own. No transaction — a mid-sequence failure leaves the camera
partially moved. Acceptable for PTZ, worth noting for the record.

### Smells
[SMELL] Global `cgiSessionMu` held across network I/O serializes logins for a service designed to
front many devices (see QUALITY.md). Per-host locking would fit the stateless multi-device design.

### Cross-layer contract (Go driver ↔ openav-mcp client.py) — invariant audit
| Invariant | Go driver | client.py | Aligned? |
|-----------|-----------|-----------|----------|
| pan range | ±162.5° | ±162.5° | ✅ |
| tilt range | -30..90° | -30..90° | ✅ |
| preset id | 0-255 | 0-255 | ✅ |
| preset name | ≤64 chars | ≤64 chars | ✅ |
| ptz speed | >0 then clamp 1-24/tilt20 | >0 | ✅ (Go clamps) |
| jog direction set | 9 tokens incl. stop | same frozenset | ✅ |
| jog speed | clamp 1-24 (tilt 20) | enforce 1-24 | ⚠ minor (tilt 21-24 clamps silently) |
| tracking mode | reject ∉{presenter,zone} | **not validated** | ⚠ drift (server schema covers MCP path only) |
| zoom | raw 16-bit position 0..0x4000 | unbounded `number`, undocumented | ⚠ semantics undocumented |

Verdict: the pan/tilt/preset/name invariants — the ones flagged historically as drift-prone — are
correctly mirrored in BOTH layers and validated in mock + live (parity fix held). Residual drift is
tracking-mode validation and zoom semantics (both logged as smells).

### Contract Compliance
| Contract | Status | Notes |
|----------|--------|-------|
| No feature contract active | N/A | `.claude/contracts/` empty — created on-demand by `/build` |

---

## Baseline (2026-03-21) — retained for history

### Blockers
_No blockers. Codebase is architecturally sound._

### Risks (baseline)
[RETIRED 2026-07-21] EC20 12-endpoint REST placeholder paths — REST layer removed; replaced by
hardware-verified VISCA + CGI planes.

[ACCEPTED] `parseSocketKey()` duplicated across both drivers — self-contained single-binary per
OpenAV convention; becomes a real risk only if a 3rd driver is added. Confirmed with Tim 2026-07-18.

### Devil's Advocate Challenges (baseline)
| File | Challenge | Verdict |
|------|-----------|---------|
| `parseSocketKey()` (both drivers) | Shared utility package? | Acceptable — OpenAV self-contained convention. |
| `ec20APIPostJSON()` | Needed? | **Moot 2026-07-21** — removed with REST layer. |
| `ec20APIGetRaw()` | Needed? | **Moot 2026-07-21** — removed with REST layer. |
| Docker `rm -f go.mod go.sum` | Why delete before init? | Required — framework submodule gitignores go.mod. |
