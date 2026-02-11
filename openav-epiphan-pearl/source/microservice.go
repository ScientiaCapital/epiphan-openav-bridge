package main

import (
	"errors"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

func setFrameworkGlobals() {
	framework.DefaultSocketPort = 80
	framework.MicroserviceName = "OpenAV Epiphan Pearl Microservice"

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
		return getDeviceStatus(socketKey)
	case "recordingstatus":
		return getRecordingStatus(socketKey)
	case "storages":
		return getStorages(socketKey)
	case "channels":
		return getChannels(socketKey)
	case "healthcheck":
		return healthCheck(socketKey)
	}

	errMsg := function + " - unrecognized setting in URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

// doDeviceSpecificSet handles all PUT requests routed by the framework.
// socketKey contains the device address with embedded credentials.
// setting is the first path parameter. arg1 is the JSON body value.
//
//	Example PUT URIs:
//	  ":address/:setting"
//	  ":address/:setting/:arg1"
//	  ":address/:setting/:arg1/:arg2"
func doDeviceSpecificSet(socketKey string, setting string, arg1 string, arg2 string, arg3 string) (string, error) {
	function := "doDeviceSpecificSet"

	framework.Log(function + " - got setting: " + setting + " arg1: " + arg1 + " arg2: " + arg2 + " arg3: " + arg3)

	switch setting {
	case "recording":
		return controlRecording(socketKey, arg1)
	case "streaming":
		return controlStreaming(socketKey, arg1, arg2)
	case "singletouch":
		return controlSingleTouch(socketKey, arg1)
	}

	errMsg := function + " - unrecognized setting in URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

func main() {
	setFrameworkGlobals()
	framework.Startup()
}
