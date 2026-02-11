package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
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

// pearlAPIGet performs an authenticated GET request to the Pearl REST API v2.0.
func pearlAPIGet(socketKey string, endpoint string) (map[string]interface{}, error) {
	function := "pearlAPIGet"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + "/api/v2.0" + endpoint

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
		errMsg := function + " - 401 Unauthorized: check Pearl credentials"
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

// pearlAPIPost performs an authenticated POST request to the Pearl REST API v2.0.
func pearlAPIPost(socketKey string, endpoint string) error {
	function := "pearlAPIPost"

	host, username, password := parseSocketKey(socketKey)
	url := "http://" + host + "/api/v2.0" + endpoint

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
		errMsg := function + " - 401 Unauthorized: check Pearl credentials"
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

// ========== P0 GET functions ==========

func getDeviceStatus(socketKey string) (string, error) {
	function := "getDeviceStatus"
	framework.Log(function + " - called for: " + socketKey)

	deviceData, err := pearlAPIGet(socketKey, "/device")
	if err != nil {
		return "", err
	}

	storageData, err := pearlAPIGet(socketKey, "/storages")
	if err != nil {
		return "", err
	}

	status := make(map[string]interface{})
	if result, ok := deviceData["result"]; ok {
		status["device"] = result
	}
	if result, ok := storageData["result"]; ok {
		status["storages"] = result
	}

	jsonBytes, err := json.Marshal(status)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling status: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func getRecordingStatus(socketKey string) (string, error) {
	function := "getRecordingStatus"
	framework.Log(function + " - called for: " + socketKey)

	data, err := pearlAPIGet(socketKey, "/recorders/status")
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		return `"unknown"`, nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func getStorages(socketKey string) (string, error) {
	function := "getStorages"
	framework.Log(function + " - called for: " + socketKey)

	data, err := pearlAPIGet(socketKey, "/storages")
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		return `"unknown"`, nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func getChannels(socketKey string) (string, error) {
	function := "getChannels"
	framework.Log(function + " - called for: " + socketKey)

	data, err := pearlAPIGet(socketKey, "/channels")
	if err != nil {
		return "", err
	}

	result, ok := data["result"]
	if !ok {
		return `"unknown"`, nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		errMsg := fmt.Sprintf(function+" - error marshaling: %v", err)
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}

	return string(jsonBytes), nil
}

func healthCheck(socketKey string) (string, error) {
	_, err := pearlAPIGet(socketKey, "/device")
	if err != nil {
		return `"false"`, nil
	}
	return `"true"`, nil
}

// ========== P0 SET functions ==========

func controlRecording(socketKey string, action string) (string, error) {
	function := "controlRecording"
	framework.Log(function + " - called for: " + socketKey + " action: " + action)

	action = strings.Trim(action, `"`)

	switch action {
	case "start":
		err := pearlAPIPost(socketKey, "/recorders/control/start")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	case "stop":
		err := pearlAPIPost(socketKey, "/recorders/control/stop")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	default:
		errMsg := function + " - invalid action: " + action + " (expected 'start' or 'stop')"
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
}

// ========== P1 SET functions ==========

func controlStreaming(socketKey string, channelID string, action string) (string, error) {
	function := "controlStreaming"
	framework.Log(function + " - called for: " + socketKey + " channel: " + channelID + " action: " + action)

	action = strings.Trim(action, `"`)
	channelID = strings.Trim(channelID, `"`)

	switch action {
	case "start":
		err := pearlAPIPost(socketKey, "/channels/"+channelID+"/publishers/control/start")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	case "stop":
		err := pearlAPIPost(socketKey, "/channels/"+channelID+"/publishers/control/stop")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	default:
		errMsg := function + " - invalid action: " + action + " (expected 'start' or 'stop')"
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
}

func controlSingleTouch(socketKey string, action string) (string, error) {
	function := "controlSingleTouch"
	framework.Log(function + " - called for: " + socketKey + " action: " + action)

	action = strings.Trim(action, `"`)

	switch action {
	case "start":
		err := pearlAPIPost(socketKey, "/singletouch/control/start")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	case "stop":
		err := pearlAPIPost(socketKey, "/singletouch/control/stop")
		if err != nil {
			return "", err
		}
		return `"ok"`, nil
	default:
		errMsg := function + " - invalid action: " + action + " (expected 'start' or 'stop')"
		framework.AddToErrors(socketKey, errMsg)
		return "", errors.New(errMsg)
	}
}
