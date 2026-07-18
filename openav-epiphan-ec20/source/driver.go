package main

import (
	"bytes"
	"encoding/base64"
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
	ec20EndpointPresets     = "/api/ptz/presets"       // PLACEHOLDER - GET preset list
	ec20EndpointPresetGoto  = "/api/ptz/preset/goto"  // PLACEHOLDER - POST {preset_id}
	ec20EndpointPresetSave  = "/api/ptz/preset/save"  // PLACEHOLDER - POST {preset_id, name}
	ec20EndpointTrackingOn  = "/api/tracking/enable"  // PLACEHOLDER - POST {mode}
	ec20EndpointTrackingOff = "/api/tracking/disable" // PLACEHOLDER - POST (no body)
	ec20EndpointPreview     = "/api/preview"           // PLACEHOLDER - GET (returns JPEG binary)
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

	var reqBody io.Reader
	if jsonBody != nil {
		reqBody = bytes.NewBuffer(jsonBody)
		framework.Log(function + " - " + method + " " + url + " body: " + string(jsonBody))
	} else {
		framework.Log(function + " - " + method + " " + url)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error creating request: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	req.SetBasicAuth(username, password)
	if jsonBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
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

func getCameraStatus(socketKey string) (string, error) {
	framework.Log("getCameraStatus - called for: " + socketKey)
	return fetchResultOrRaw(socketKey, "getCameraStatus", "status", ec20EndpointStatus)
}

func getPTZPosition(socketKey string) (string, error) {
	framework.Log("getPTZPosition - called for: " + socketKey)
	return fetchResultOrRaw(socketKey, "getPTZPosition", "position", ec20EndpointPosition)
}

func getPresets(socketKey string) (string, error) {
	framework.Log("getPresets - called for: " + socketKey)
	return fetchResultOrRaw(socketKey, "getPresets", "presets", ec20EndpointPresets)
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
	_, err := ec20APIGet(socketKey, ec20EndpointStatus)
	if err != nil {
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

	// Pan
	_, err = ec20APIPostJSON(socketKey, ec20EndpointPan, map[string]interface{}{
		"degrees": pan,
		"speed":   speed,
	})
	if err != nil {
		return "", err
	}

	// Tilt
	_, err = ec20APIPostJSON(socketKey, ec20EndpointTilt, map[string]interface{}{
		"degrees": tilt,
		"speed":   speed,
	})
	if err != nil {
		return "", err
	}

	// Zoom
	_, err = ec20APIPostJSON(socketKey, ec20EndpointZoom, map[string]interface{}{
		"level": zoom,
	})
	if err != nil {
		return "", err
	}

	return `"ok"`, nil
}

func controlPTZHome(socketKey string) (string, error) {
	function := "controlPTZHome"
	framework.Log(function + " - called for: " + socketKey)

	err := ec20APIPost(socketKey, ec20EndpointHome)
	if err != nil {
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

	_, err := ec20APIPostJSON(socketKey, ec20EndpointPresetGoto, map[string]interface{}{
		"preset_id": presetID,
	})
	if err != nil {
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

	_, err := ec20APIPostJSON(socketKey, ec20EndpointPresetSave, map[string]interface{}{
		"preset_id": presetID,
		"name":      name,
	})
	if err != nil {
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
		_, err := ec20APIPostJSON(socketKey, ec20EndpointTrackingOn, map[string]interface{}{
			"mode": mode,
		})
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	case "disable":
		err := ec20APIPost(socketKey, ec20EndpointTrackingOff)
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	default:
		errMsg := function + " - invalid action: " + action + " (expected 'enable' or 'disable')"
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
}
