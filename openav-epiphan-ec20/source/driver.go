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
// All paths are PLACEHOLDER values. The EC20 REST API is not publicly documented.
// Update this block when endpoints are confirmed on real hardware.

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

// ec20APIGet performs an authenticated GET request to the EC20 REST API.
// Unlike Pearl, EC20 endpoints include the full path in constants (no /api/v2.0 prefix).
// Tolerates responses without a "status" field since the EC20 response format is unknown.
func ec20APIGet(socketKey string, endpoint string) (map[string]interface{}, error) {
	function := "ec20APIGet"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + endpoint

	framework.Log(function + " - GET " + url)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error creating request: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error doing GET: %v", err)
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

	framework.Log(function + " - response: " + string(bodyBytes))

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

	var result map[string]interface{}
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error parsing JSON: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	// Tolerate responses without "status" field — EC20 response format is unknown
	if status, ok := result["status"].(string); ok && status != "ok" {
		msg := ""
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		errMsg := fmt.Sprintf(function+" - API error: %s - %s", status, msg)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	return result, nil
}

// ec20APIPost performs an authenticated body-less POST request to the EC20 REST API.
// Used for endpoints like home and tracking/disable that take no body.
func ec20APIPost(socketKey string, endpoint string) error {
	function := "ec20APIPost"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + endpoint

	framework.Log(function + " - POST " + url)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error creating request: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error doing POST: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error reading response: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}

	framework.Log(function + " - response: " + string(bodyBytes))

	if resp.StatusCode == http.StatusUnauthorized {
		errMsg := function + " - 401 Unauthorized: check EC20 credentials"
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf(function+" - HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		framework.AddToErrors(socketKey, errMsg)
		return errors.New(errMsg)
	}

	var result map[string]interface{}
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		// Some POST endpoints may not return JSON - that's OK
		return nil
	}

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

// ec20APIPostJSON performs an authenticated POST request with a JSON body.
// Used for PTZ commands, preset operations, and tracking enable that require parameters.
func ec20APIPostJSON(socketKey string, endpoint string, body map[string]interface{}) (map[string]interface{}, error) {
	function := "ec20APIPostJSON"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + endpoint

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling JSON body: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	framework.Log(function + " - POST " + url + " body: " + string(jsonBytes))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error creating request: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error doing POST: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error reading response: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	framework.Log(function + " - response: " + string(respBytes))

	if resp.StatusCode == http.StatusUnauthorized {
		errMsg := function + " - 401 Unauthorized: check EC20 credentials"
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf(function+" - HTTP %d: %s", resp.StatusCode, string(respBytes))
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBytes, &result)
	if err != nil {
		// Some POST endpoints may not return JSON - return empty map
		return make(map[string]interface{}), nil
	}

	if status, ok := result["status"].(string); ok && status != "ok" {
		msg := ""
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		errMsg := fmt.Sprintf(function+" - API error: %s - %s", status, msg)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}

	return result, nil
}

// ec20APIGetRaw performs an authenticated GET request and returns the raw response bytes.
// Used by getPreview for JPEG binary data — no JSON parsing attempted.
func ec20APIGetRaw(socketKey string, endpoint string) ([]byte, error) {
	function := "ec20APIGetRaw"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + endpoint

	framework.Log(function + " - GET " + url)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error creating request: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return nil, errors.New(errMsg)
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error doing GET: %v", err)
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

// ========== GET functions ==========

func getCameraStatus(socketKey string) (string, error) {
	function := "getCameraStatus"
	framework.Log(function + " - called for: " + socketKey)

	data, err := ec20APIGet(socketKey, ec20EndpointStatus)
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
		errMsg := fmt.Sprintf(function+" - error marshaling status: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func getPTZPosition(socketKey string) (string, error) {
	function := "getPTZPosition"
	framework.Log(function + " - called for: " + socketKey)

	data, err := ec20APIGet(socketKey, ec20EndpointPosition)
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		result = data
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling position: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func getPresets(socketKey string) (string, error) {
	function := "getPresets"
	framework.Log(function + " - called for: " + socketKey)

	data, err := ec20APIGet(socketKey, ec20EndpointPresets)
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		result = data
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling presets: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
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

func controlPTZ(socketKey string, panStr string, tiltStr string, zoomStr string) (string, error) {
	function := "controlPTZ"
	framework.Log(function + " - called for: " + socketKey + " pan: " + panStr + " tilt: " + tiltStr + " zoom: " + zoomStr)

	panStr = strings.Trim(panStr, `"`)
	tiltStr = strings.Trim(tiltStr, `"`)
	zoomStr = strings.Trim(zoomStr, `"`)

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

	zoom, err := strconv.ParseFloat(zoomStr, 64)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - invalid zoom value: %s", zoomStr)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	// Pan
	_, err = ec20APIPostJSON(socketKey, ec20EndpointPan, map[string]interface{}{
		"degrees": pan,
		"speed":   50,
	})
	if err != nil {
		return "", err
	}

	// Tilt
	_, err = ec20APIPostJSON(socketKey, ec20EndpointTilt, map[string]interface{}{
		"degrees": tilt,
		"speed":   50,
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
