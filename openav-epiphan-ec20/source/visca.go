package main

// VISCA-over-IP client for the EC20 PTZ/preset plane.
//
// Epiphan sanctions VISCA-over-IP (fixed UDP :52381) as an EC20 integration path
// (must be enabled on the device). It is a stable, standardized binary protocol
// for pan/tilt/zoom, presets and absolute positioning — used here for everything
// EXCEPT AI-tracking (which is CGI-only; see cgiauth.go / driver.go).
//
// Sony VISCA-over-IP framing: an 8-byte header precedes each VISCA payload:
//   [payloadType(2)] [payloadLength(2)] [sequenceNumber(4)]   (all big-endian)
// Camera messages are addressed to camera 1, so the VISCA payload starts 0x81
// and ends 0xFF. Replies: ACK 0x90 0x4y 0xFF, completion 0x90 0x5y 0xFF.

import (
	"encoding/binary"
)

const (
	viscaDefaultPort = "52381"
	viscaTypeCommand = 0x0100 // VISCA command payload
	viscaTypeInquiry = 0x0110 // VISCA inquiry payload
	viscaHeaderByte  = 0x81   // controller 0 -> camera 1
)

// Jog direction bytes (VISCA pan-tilt drive).
const (
	viscaPanLeft   byte = 0x01
	viscaPanRight  byte = 0x02
	viscaPanStop   byte = 0x03
	viscaTiltUp    byte = 0x01
	viscaTiltDown  byte = 0x02
	viscaTiltStop  byte = 0x03
)

// viscaWrap prepends the Sony VISCA-over-IP 8-byte header to a VISCA payload:
// [payloadType(2)] [payloadLength(2)] [sequenceNumber(4)], all big-endian.
func viscaWrap(payloadType uint16, seq uint32, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint16(out[0:2], payloadType)
	binary.BigEndian.PutUint16(out[2:4], uint16(len(payload)))
	binary.BigEndian.PutUint32(out[4:8], seq)
	copy(out[8:], payload)
	return out
}

// viscaPresetRecall builds "recall preset n": 81 01 04 3F 02 0n FF.
func viscaPresetRecall(preset int) []byte {
	return []byte{viscaHeaderByte, 0x01, 0x04, 0x3F, 0x02, byte(preset), 0xFF}
}

// viscaPresetSet builds "store preset n": 81 01 04 3F 01 0n FF.
func viscaPresetSet(preset int) []byte {
	return []byte{viscaHeaderByte, 0x01, 0x04, 0x3F, 0x01, byte(preset), 0xFF}
}

// viscaHome builds "pan-tilt home": 81 01 06 04 FF.
func viscaHome() []byte {
	return []byte{viscaHeaderByte, 0x01, 0x06, 0x04, 0xFF}
}

// viscaJog builds "pan-tilt drive": 81 01 06 01 VV WW <panDir> <tiltDir> FF.
func viscaJog(panDir, tiltDir byte, panSpeed, tiltSpeed int) []byte {
	return []byte{viscaHeaderByte, 0x01, 0x06, 0x01,
		byte(panSpeed), byte(tiltSpeed), panDir, tiltDir, 0xFF}
}

// viscaStop builds a pan-tilt stop: 81 01 06 01 VV WW 03 03 FF.
func viscaStop(panSpeed, tiltSpeed int) []byte {
	return viscaJog(viscaPanStop, viscaTiltStop, panSpeed, tiltSpeed)
}
