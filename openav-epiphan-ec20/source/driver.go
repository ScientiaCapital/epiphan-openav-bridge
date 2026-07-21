package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

// ========== EC20 API Endpoint Constants ==========
// The REST URL paths below are PLACEHOLDER values. The EC20's REST API paths are NOT
// published in Epiphan's public docs (the AI User Guide, Q-SYS plugin README, and tech
// specs all omit them — see .claude/programs/ec20-api-discovery.md, 2026-07-17 research).
// Confirm them on real hardware with ec20_probe.sh, then update this block.
//
// NOTE: while the *paths* are unverified, the EC20's control *behavior* IS documented and
// is enforced elsewhere in this file (DOC-CONFIRMED 2026-07-17): preset range 0-255,
// tracking modes presenter/zone, pan ±162.5°, tilt -30°..+90°, HTTP port 80, Basic Auth.

const (
	ec20EndpointStatus      = "/api/status"           // PLACEHOLDER - GET device status
	ec20EndpointPosition    = "/api/ptz/position"     // PLACEHOLDER - GET current PTZ position
	ec20EndpointPan         = "/api/ptz/pan"          // PLACEHOLDER - POST {degrees, speed}
	ec20EndpointTilt        = "/api/ptz/tilt"         // PLACEHOLDER - POST {degrees, speed}
	ec20EndpointZoom        = "/api/ptz/zoom"         // PLACEHOLDER - POST {level}
	ec20EndpointHome        = "/api/ptz/home"         // PLACEHOLDER - POST (no body)
	ec20EndpointPresets     = "/api/ptz/presets"      // PLACEHOLDER - GET preset list
	ec20EndpointPresetGoto  = "/api/ptz/preset/goto"  // PLACEHOLDER - POST {preset_id}
	ec20EndpointPresetSave  = "/api/ptz/preset/save"  // PLACEHOLDER - POST {preset_id, name}
	ec20EndpointTrackingOn  = "/api/tracking/enable"  // PLACEHOLDER - POST {mode}
	ec20EndpointTrackingOff = "/api/tracking/disable" // PLACEHOLDER - POST (no body)
	ec20EndpointPreview     = "/api/preview"          // PLACEHOLDER - GET (returns JPEG binary)
)

// parseSocketKey extracts host, username, and password from the framework socketKey.
// socketKey format: "[protocol|]username:password@host:port"
func parseSocketKey(socketKey string) (host string, username string, password string) {
	key := framework.StripProtocolPrefix(socketKey)

	if strings.Contains(key, "@") {
		parts := strings.SplitN(key, "@", 2)
		host = parts[1]
		creds := parts[0]
		if strings.Contains(creds, ":") {
			credParts := strings.SplitN(creds, ":", 2)
			username = credParts[0]
			password = credParts[1]
		}
	} else {
		host = key
	}

	return
}

// validatePresetID ensures presetID is a valid integer in the EC20 preset range 0-255.
// Preset 0 IS valid: the EC20 User Guide notes "If preset 0 is saved, PTZ will be moved
// to preset 0" during init, and the tech specs list a maximum of 255 presets.
// (DOC-CONFIRMED 2026-07-17 — previously this rejected preset 0, a bug.)
func validatePresetID(presetID string) error {
	id, err := strconv.Atoi(presetID)
	if err != nil {
		return fmt.Errorf("presetID must be numeric: %s", presetID)
	}
	if id < 0 || id > 255 {
		return fmt.Errorf("presetID out of range (0-255): %d", id)
	}
	return nil
}

// ec20APIRequest performs an authenticated HTTP request to the EC20 REST API and returns the
// raw response body. Shared by ec20APIGet/ec20APIPost/ec20APIPostJSON/ec20APIGetRaw.
// logResponse=false skips logging the body (used for raw binary responses, e.g. JPEG preview,
// where logging it as a string would be useless/huge).
func ec20APIRequest(socketKey, function, method, endpoint string, jsonBody []byte, logResponse bool) ([]byte, error) {
	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + endpoint

	if jsonBody != nil {
		framework.Log(function + " - " + method + " " + url + " body: " + string(jsonBody))
	} else {
		framework.Log(function + " - " + method + " " + url)
	}

	// The EC20 (lighttpd) requires HTTP Digest auth, not Basic. ec20DoWithDigest
	// performs the request, transparently answering a Digest challenge; it falls
	// back to Basic for servers that don't advertise Digest.
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := ec20DoWithDigest(client, method, url, jsonBody, username, password)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error doing %s: %v", method, err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error reading response: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	if logResponse {
		framework.Log(function + " - response: " + string(bodyBytes))
	}

	if resp.StatusCode == http.StatusUnauthorized {
		errMsg := function + " - 401 Unauthorized: check EC20 credentials"
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf(function+" - HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	return bodyBytes, nil
}

// ec20NewRequest builds a request with an optional JSON body, setting the
// Content-Type header when a body is present. The body is rebuilt from jsonBody
// on each call so a request can be safely re-sent (needed for the Digest retry).
func ec20NewRequest(method, url string, jsonBody []byte) (*http.Request, error) {
	var body io.Reader
	if jsonBody != nil {
		body = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if jsonBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// ec20DoWithDigest performs an HTTP request against the EC20, transparently
// handling an HTTP Digest challenge. The real EC20 (lighttpd/1.4.75) answers with
// "WWW-Authenticate: Digest ... algorithm=MD5, qop=auth" and rejects Basic auth,
// so a Basic-only client 401s regardless of correct credentials. Go's stdlib has
// no Digest client; this implements RFC 2617 (MD5, qop=auth) with only crypto/md5
// and crypto/rand — no external dependency, preserving the Docker/GPL-3.0 posture.
// If the server does not advertise Digest, it falls back to Basic auth.
func ec20DoWithDigest(client *http.Client, method, url string, jsonBody []byte, username, password string) (*http.Response, error) {
	// First attempt with no credentials — obtain the challenge (or a direct 2xx).
	req, err := ec20NewRequest(method, url, jsonBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil // no authentication required
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()

	req2, err := ec20NewRequest(method, url, jsonBody)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(challenge)), "digest") {
		header, err := ec20DigestAuthHeader(challenge, method, req2.URL.RequestURI(), username, password)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("Authorization", header)
	} else {
		req2.SetBasicAuth(username, password)
	}
	return client.Do(req2)
}

// ec20DigestAuthHeader builds an RFC 2617 Digest "Authorization" header for the
// given challenge, request method and URI. Supports algorithm=MD5 with qop="auth"
// (and the legacy qop-absent form).
func ec20DigestAuthHeader(challenge, method, uri, username, password string) (string, error) {
	p := parseDigestChallenge(challenge)
	realm, nonce, qop, opaque := p["realm"], p["nonce"], p["qop"], p["opaque"]

	ha1 := md5Hex(username + ":" + realm + ":" + password)
	ha2 := md5Hex(method + ":" + uri)

	var b strings.Builder
	fmt.Fprintf(&b, `Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm=MD5`,
		username, realm, nonce, uri)

	var response string
	if strings.Contains(qop, "auth") {
		cnonce, err := ec20Cnonce()
		if err != nil {
			return "", err
		}
		const nc = "00000001"
		response = md5Hex(strings.Join([]string{ha1, nonce, nc, cnonce, "auth", ha2}, ":"))
		fmt.Fprintf(&b, `, qop=auth, nc=%s, cnonce="%s"`, nc, cnonce)
	} else {
		response = md5Hex(ha1 + ":" + nonce + ":" + ha2)
	}
	fmt.Fprintf(&b, `, response="%s"`, response)
	if opaque != "" {
		fmt.Fprintf(&b, `, opaque="%s"`, opaque)
	}
	return b.String(), nil
}

// parseDigestChallenge parses a "Digest k=v, ..." WWW-Authenticate header into a
// map, splitting on commas that are not inside quoted values.
func parseDigestChallenge(challenge string) map[string]string {
	body := strings.TrimSpace(challenge)
	if i := strings.IndexByte(body, ' '); i >= 0 && strings.EqualFold(body[:i], "Digest") {
		body = body[i+1:]
	}
	out := map[string]string{}
	var cur strings.Builder
	inQuotes := false
	flush := func() {
		part := strings.TrimSpace(cur.String())
		cur.Reset()
		if part == "" {
			return
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		out[key] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}
	for _, r := range body {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			cur.WriteRune(r)
		case r == ',' && !inQuotes:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// md5Hex returns the lowercase hex MD5 of s (the hash primitive for RFC 2617).
func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ec20Cnonce returns a random client nonce for the Digest handshake.
func ec20Cnonce() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// checkAPIStatus returns an error if result carries a tolerant "status" field != "ok".
// Tolerates responses without a "status" field since the EC20 response format is unknown.
func checkAPIStatus(socketKey, function string, result map[string]interface{}) error {
	if status, ok := result["status"].(string); ok && status != "ok" {
		msg := ""
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		errMsg := fmt.Sprintf(function+" - API error: %s - %s", status, msg)
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// ec20APIGet performs an authenticated GET request to the EC20 REST API.
// Unlike Pearl, EC20 endpoints include the full path in constants (no /api/v2.0 prefix).
func ec20APIGet(socketKey string, endpoint string) (map[string]interface{}, error) {
	function := "ec20APIGet"

	bodyBytes, err := ec20APIRequest(socketKey, function, "GET", endpoint, nil, true)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		errMsg := fmt.Sprintf(function+" - error parsing JSON: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	if err := checkAPIStatus(socketKey, function, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ec20APIPost performs an authenticated body-less POST request to the EC20 REST API.
// Used for endpoints like home and tracking/disable that take no body.
func ec20APIPost(socketKey string, endpoint string) error {
	function := "ec20APIPost"

	bodyBytes, err := ec20APIRequest(socketKey, function, "POST", endpoint, nil, true)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// Some POST endpoints may not return JSON - that's OK
		return nil
	}

	return checkAPIStatus(socketKey, function, result)
}

// ec20APIPostJSON performs an authenticated POST request with a JSON body.
// Used for PTZ commands, preset operations, and tracking enable that require parameters.
func ec20APIPostJSON(socketKey string, endpoint string, body map[string]interface{}) (map[string]interface{}, error) {
	function := "ec20APIPostJSON"

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling JSON body: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	respBytes, err := ec20APIRequest(socketKey, function, "POST", endpoint, jsonBytes, true)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		// Some POST endpoints may not return JSON - return empty map
		return make(map[string]interface{}), nil
	}

	if err := checkAPIStatus(socketKey, function, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ec20APIGetRaw performs an authenticated GET request and returns the raw response bytes.
// Used by getPreview for JPEG binary data — no JSON parsing attempted.
func ec20APIGetRaw(socketKey string, endpoint string) ([]byte, error) {
	return ec20APIRequest(socketKey, "ec20APIGetRaw", "GET", endpoint, nil, false)
}

// ========== GET functions ==========

// fetchResultOrRaw calls ec20APIGet(endpoint) and marshals the "result" field, falling back to
// the full response if EC20 returns data at the top level (shape unknown pending hardware) —
// the shared shape of getCameraStatus/getPTZPosition/getPresets. fieldLabel feeds the
// per-function error text (e.g. "error marshaling status: %v").
func fetchResultOrRaw(socketKey, function, fieldLabel, endpoint string) (string, error) {
	data, err := ec20APIGet(socketKey, endpoint)
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		// EC20 may return data at top level rather than nested under "result"
		result = data
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling "+fieldLabel+": %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

// readPTZUnits queries pan/tilt and zoom via VISCA inquiries, returning RAW VISCA
// units (degrees require the Story-D calibration; until then we surface units so
// the data stays honest).
func readPTZUnits(socketKey string) (pan, tilt int16, zoom uint16, err error) {
	ptReply, err := viscaSend(socketKey, viscaPanTiltInquiry())
	if err != nil {
		return 0, 0, 0, err
	}
	if pan, tilt, err = parsePanTiltReply(ptReply); err != nil {
		return 0, 0, 0, err
	}
	zReply, err := viscaSend(socketKey, viscaZoomInquiry())
	if err != nil {
		return 0, 0, 0, err
	}
	if zoom, err = parseZoomReply(zReply); err != nil {
		return 0, 0, 0, err
	}
	return pan, tilt, zoom, nil
}

func getCameraStatus(socketKey string) (string, error) {
	function := "getCameraStatus"
	framework.Log(function + " - called for: " + socketKey)
	pan, tilt, zoom, err := readPTZUnits(socketKey)
	if err != nil {
		framework.AddToErrors(socketKey, function+" - "+err.Error())
		return "", err
	}
	out, _ := json.Marshal(map[string]interface{}{
		"online": true, "pan_units": pan, "tilt_units": tilt, "zoom_units": zoom,
	})
	return string(out), nil
}

func getPTZPosition(socketKey string) (string, error) {
	function := "getPTZPosition"
	framework.Log(function + " - called for: " + socketKey)
	pan, tilt, zoom, err := readPTZUnits(socketKey)
	if err != nil {
		framework.AddToErrors(socketKey, function+" - "+err.Error())
		return "", err
	}
	out, _ := json.Marshal(map[string]interface{}{
		"pan_units": pan, "tilt_units": tilt, "zoom_units": zoom,
	})
	return string(out), nil
}

func getPresets(socketKey string) (string, error) {
	framework.Log("getPresets - called for: " + socketKey)
	// VISCA has no "list presets" inquiry; presets are addressed by slot (recall/set).
	return `"unsupported: VISCA has no list-presets inquiry; recall/set by slot 0-255"`, nil
}

func getPreview(socketKey string) (string, error) {
	function := "getPreview"
	framework.Log(function + " - called for: " + socketKey)

	rawBytes, err := ec20APIGetRaw(socketKey, ec20EndpointPreview)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(rawBytes)
	jsonBytes, err := json.Marshal(encoded)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling preview: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func healthCheck(socketKey string) (string, error) {
	// A VISCA version inquiry is the cheapest liveness probe on the control plane.
	if _, err := viscaSend(socketKey, viscaVersionInquiry()); err != nil {
		return `"false"`, nil
	}
	return `"true"`, nil
}

// ========== SET functions ==========

// ptzBody is the PUT .../ptz/:pan/:tilt request body: {"zoom": <number>, "speed": <optional int>}.
// speed defaults to 50 (the prior hardcoded value) when omitted.
type ptzBody struct {
	Zoom  *float64 `json:"zoom"`
	Speed *int     `json:"speed"`
}

// ---------- degrees/zoom -> VISCA-unit calibration ----------
//
// The MCP contract speaks degrees (pan ±162.5°, tilt -30..90°) and a zoom level;
// VISCA absolute-position frames speak raw signed 16-bit units. These constants
// map between them.
//
// !!! CALIBRATE ON HARDWARE (Live Room v1, Story D) !!!  The scales below are
// PLACEHOLDERS derived from one live data point (a 0.6s pan jog @ speed 6 read
// back as ~0x0052 units) and the documented pan/tilt ranges — NOT yet confirmed
// against absolute-position moves. Drive to known references, read the position
// inquiry, and set these to the measured units-per-degree before trusting
// absolute positioning. Values are isolated here so calibration is a one-line tune.
const (
	panUnitsPerDegree  = 14.0 // PLACEHOLDER — measure on hardware (Story D)
	tiltUnitsPerDegree = 14.0 // PLACEHOLDER — measure on hardware (Story D)
	// Zoom level range: the EC20 zoom-direct value is a 16-bit position
	// (0x0000 wide .. 0x4000 tele observed). The contract's zoom is passed as a
	// raw VISCA zoom position, clamped to this range. (NEEDS-PROBE for the exact
	// tele maximum; 0x4000 is the Sony convention.)
	zoomMax = 0x4000
	// defaultPTZSpeed feeds the VISCA absolute-move pan/tilt speed bytes when the
	// caller omits speed (matches the prior REST default of 50, clamped to VISCA's
	// documented pan-speed ceiling).
	viscaPanSpeedMax  = 0x18 // 24
	viscaTiltSpeedMax = 0x14 // 20
)

// degreesToUnits converts a degree value to a signed 16-bit VISCA position,
// clamped to the int16 range so a bad scale can never wrap the frame.
func degreesToUnits(deg, unitsPerDegree float64) int16 {
	v := deg * unitsPerDegree
	if v > 32767 {
		v = 32767
	} else if v < -32768 {
		v = -32768
	}
	return int16(v)
}

// clampSpeed maps a caller PTZ speed onto VISCA's per-axis speed bytes.
func clampSpeed(speed, max int) byte {
	if speed < 1 {
		speed = 1
	}
	if speed > max {
		speed = max
	}
	return byte(speed)
}

// zoomToUnits clamps a contract zoom level to the VISCA 16-bit zoom range.
func zoomToUnits(zoom float64) uint16 {
	if zoom < 0 {
		zoom = 0
	}
	if zoom > zoomMax {
		zoom = zoomMax
	}
	return uint16(zoom)
}

func controlPTZ(socketKey string, panStr string, tiltStr string, bodyStr string) (string, error) {
	function := "controlPTZ"
	framework.Log(function + " - called for: " + socketKey + " pan: " + panStr + " tilt: " + tiltStr + " body: " + bodyStr)

	panStr = strings.Trim(panStr, `"`)
	tiltStr = strings.Trim(tiltStr, `"`)

	pan, err := strconv.ParseFloat(panStr, 64)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - invalid pan value: %s", panStr)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	tilt, err := strconv.ParseFloat(tiltStr, 64)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - invalid tilt value: %s", tiltStr)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	var body ptzBody
	if jsonErr := json.Unmarshal([]byte(bodyStr), &body); jsonErr != nil || body.Zoom == nil {
		errMsg := fmt.Sprintf(function+` - invalid zoom value (expected JSON body {"zoom":<num>,"speed":<optional int>}): %s`, bodyStr)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
	zoom := *body.Zoom

	// speed is a caller-settable PTZ move speed; the EC20 API accepts it per-axis but has no
	// documented valid range, so only guard against non-positive values. Defaults to the
	// previously-hardcoded 50 when the caller doesn't specify one.
	speed := 50
	if body.Speed != nil {
		speed = *body.Speed
	}
	if speed <= 0 {
		errMsg := fmt.Sprintf(function+" - speed must be positive: %d", speed)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	// Enforce DOC-CONFIRMED physical limits (EC20 User Guide, PTZ specs 2026-07-17):
	// pan ±162.5°, tilt -30°..+90°. Zoom has no documented absolute scale, so it is
	// passed through unvalidated (NEEDS-PROBE).
	if pan < -162.5 || pan > 162.5 {
		errMsg := fmt.Sprintf(function+" - pan out of range (-162.5..162.5): %v", pan)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
	if tilt < -30 || tilt > 90 {
		errMsg := fmt.Sprintf(function+" - tilt out of range (-30..90): %v", tilt)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	// Absolute pan+tilt is a single VISCA frame (81 01 06 02 ...). Convert the
	// validated degrees to VISCA units via the Story-D calibration constants.
	panSpeed := clampSpeed(speed, viscaPanSpeedMax)
	tiltSpeed := clampSpeed(speed, viscaTiltSpeedMax)
	panUnits := degreesToUnits(pan, panUnitsPerDegree)
	tiltUnits := degreesToUnits(tilt, tiltUnitsPerDegree)
	if _, err := viscaSend(socketKey, viscaPanTiltAbsolute(panSpeed, tiltSpeed, panUnits, tiltUnits)); err != nil {
		framework.AddToErrors(socketKey, function+" - pan/tilt: "+err.Error())
		return "", err
	}

	// Zoom is a separate VISCA frame (81 01 04 47 ...).
	if _, err := viscaSend(socketKey, viscaZoomDirect(zoomToUnits(zoom))); err != nil {
		framework.AddToErrors(socketKey, function+" - zoom: "+err.Error())
		return "", err
	}

	return `"ok"`, nil
}

func controlPTZHome(socketKey string) (string, error) {
	function := "controlPTZHome"
	framework.Log(function + " - called for: " + socketKey)

	if _, err := viscaSend(socketKey, viscaHome()); err != nil {
		framework.AddToErrors(socketKey, function+" - "+err.Error())
		return "", err
	}

	return `"ok"`, nil
}

func recallPreset(socketKey string, presetID string) (string, error) {
	function := "recallPreset"
	framework.Log(function + " - called for: " + socketKey + " preset: " + presetID)

	presetID = strings.Trim(presetID, `"`)

	if err := validatePresetID(presetID); err != nil {
		errMsg := function + " - " + err.Error()
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
	id, _ := strconv.Atoi(presetID) // safe: validatePresetID confirmed 0-255

	if _, err := viscaSend(socketKey, viscaPresetRecall(id)); err != nil {
		framework.AddToErrors(socketKey, function+" - "+err.Error())
		return "", err
	}

	return `"ok"`, nil
}

func savePreset(socketKey string, presetID string, name string) (string, error) {
	function := "savePreset"
	framework.Log(function + " - called for: " + socketKey + " preset: " + presetID + " name: " + name)

	presetID = strings.Trim(presetID, `"`)
	name = strings.Trim(name, `"`)

	if err := validatePresetID(presetID); err != nil {
		errMsg := function + " - " + err.Error()
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	if len(name) > 64 {
		errMsg := fmt.Sprintf(function+" - preset name too long: %d chars (max 64)", len(name))
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
	id, _ := strconv.Atoi(presetID) // safe: validatePresetID confirmed 0-255

	// VISCA "store preset" carries only the slot number — the name is not sent to
	// the camera (it's validated above for API compatibility and future use).
	if _, err := viscaSend(socketKey, viscaPresetSet(id)); err != nil {
		framework.AddToErrors(socketKey, function+" - "+err.Error())
		return "", err
	}

	return `"ok"`, nil
}

func controlTracking(socketKey string, action string, mode string) (string, error) {
	function := "controlTracking"
	framework.Log(function + " - called for: " + socketKey + " action: " + action + " mode: " + mode)

	action = strings.Trim(action, `"`)
	mode = strings.Trim(mode, `"`)

	switch action {
	case "enable":
		// Tracking modes are DOC-CONFIRMED (EC20 User Guide, Tracking Configuration):
		// "Presenter" (default, aka Human Tracking) and "Zone". Default to presenter
		// when unspecified, and reject anything else.
		mode = strings.ToLower(mode)
		if mode == "" {
			mode = "presenter"
		}
		if mode != "presenter" && mode != "zone" {
			errMsg := function + " - invalid tracking mode: " + mode + " (expected 'presenter' or 'zone')"
			framework.AddToErrors(socketKey, errMsg)
			return "", errors.New(errMsg)
		}
		// AI tracking is not part of standard VISCA — it rides the device's CGI
		// surface (auth.cgi session + ptzctrl.cgi command; see cgiauth.go).
		if _, err := ec20CGISendGET(socketKey, ec20TrackingCommand("enable", mode)); err != nil {
			framework.AddToErrors(socketKey, function+" - "+err.Error())
			return "", err
		}
		return `"ok"`, nil
	case "disable":
		if _, err := ec20CGISendGET(socketKey, ec20TrackingCommand("disable", "")); err != nil {
			framework.AddToErrors(socketKey, function+" - "+err.Error())
			return "", err
		}
		return `"ok"`, nil
	default:
		errMsg := function + " - invalid action: " + action + " (expected 'enable' or 'disable')"
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
}
