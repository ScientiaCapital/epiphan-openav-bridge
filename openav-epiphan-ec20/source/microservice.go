package main

import (
	"errors"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

func setFrameworkGlobals() {
	framework.DefaultSocketPort = 80
	framework.MicroserviceName = "OpenAV Epiphan EC20 Microservice"

	framework.RegisterMainGetFunc(doDeviceSpecificGet)
	framework.RegisterMainSetFunc(doDeviceSpecificSet)
}

// doDeviceSpecificGet handles all GET requests routed by the framework.
// socketKey contains the device address with embedded credentials.
// setting is the first path parameter after the address.
//
//	Example GET URIs:
//	  ":address/:setting"
//	  ":address/:setting/:arg1"
//	  ":address/:setting/:arg1/:arg2"
func doDeviceSpecificGet(socketKey string, setting string, arg1 string, arg2 string) (string, error) {
	function := "doDeviceSpecificGet"

	framework.Log(function + " - got setting: " + setting + " arg1: " + arg1 + " arg2: " + arg2)

	switch setting {
	case "status":
		return getCameraStatus(socketKey)
	case "healthcheck":
		return healthCheck(socketKey)
	case "ptzposition":
		return getPTZPosition(socketKey)
	case "presets":
		return getPresets(socketKey)
	case "preview":
		return getPreview(socketKey)
	}

	errMsg := function + " - unrecognized setting in URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

// doDeviceSpecificSet handles all PUT requests routed by the framework.
// socketKey contains the device address with embedded credentials.
// setting is the first path parameter. arg1/arg2 are URL path params. arg3 is the JSON body.
//
//	Example PUT URIs:
//	  ":address/:setting"               → setting=X, arg1=body, arg2="", arg3=""
//	  ":address/:setting/:arg1"         → setting=X, arg1=Y,    arg2=body, arg3=""
//	  ":address/:setting/:arg1/:arg2"   → setting=X, arg1=Y,    arg2=Z,    arg3=body
func doDeviceSpecificSet(socketKey string, setting string, arg1 string, arg2 string, arg3 string) (string, error) {
	function := "doDeviceSpecificSet"

	framework.Log(function + " - got setting: " + setting + " arg1: " + arg1 + " arg2: " + arg2 + " arg3: " + arg3)

	switch setting {
	case "ptz":
		// PUT /:addr/ptz/:pan/:tilt  body=zoom
		return controlPTZ(socketKey, arg1, arg2, arg3)
	case "ptzhome":
		// PUT /:addr/ptzhome
		return controlPTZHome(socketKey)
	case "preset":
		// PUT /:addr/preset/:presetId  body=""
		return recallPreset(socketKey, arg1)
	case "presetsave":
		// PUT /:addr/presetsave/:presetId  body=name
		return savePreset(socketKey, arg1, arg2)
	case "tracking":
		// PUT /:addr/tracking/:action  body=mode
		return controlTracking(socketKey, arg1, arg2)
	}

	errMsg := function + " - unrecognized setting in URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

func main() {
	setFrameworkGlobals()
	framework.Startup()
}
