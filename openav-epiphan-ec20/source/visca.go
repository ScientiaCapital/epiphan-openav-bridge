package main

// VISCA-over-IP client for the EC20 PTZ/preset/zoom plane.
//
// TRANSPORT (hardware-verified 2026-07-18/19 against a real EC20, fw 3.3.40):
// the EC20 speaks **raw VISCA over TCP :5678** — plain VISCA payload bytes, NO
// Sony 8-byte UDP wrapper. A version inquiry `81 09 00 02 FF` returns
// `90 50 00 52 FF`, and pan/tilt/zoom/home/preset commands return ACK `90 4x FF`
// then completion `90 5x FF`. (The Epiphan docs list Sony VISCA-over-IP on UDP
// :52381, but that path did not respond on this unit; TCP :5678 raw is what
// works out-of-box and is used here.) Everything EXCEPT AI-tracking rides this
// plane; tracking is CGI-only (see cgiauth.go).
//
// VISCA framing: controller→camera messages start 0x81 (controller 0 → camera 1)
// and end 0xFF. Replies: ACK 0x90 0x4y 0xFF, completion 0x90 0x5y 0xFF, error
// 0x90 0x6y 0xFF. Inquiry replies are a completion carrying data: 0x90 0x50 … 0xFF.

import (
	"fmt"
	"net"
	"time"
)

const viscaHeaderByte = 0x81 // controller 0 -> camera 1

// viscaPort is the TCP port the EC20 exposes raw VISCA on. It is a var (not a
// const) so tests can point viscaSend at a fake listener on an ephemeral port.
var viscaPort = "5678"

// viscaDialTimeout bounds both the TCP connect and each read/write.
var viscaDialTimeout = 5 * time.Second

// Jog direction bytes (VISCA pan-tilt drive).
const (
	viscaPanLeft  byte = 0x01
	viscaPanRight byte = 0x02
	viscaPanStop  byte = 0x03
	viscaTiltUp   byte = 0x01
	viscaTiltDown byte = 0x02
	viscaTiltStop byte = 0x03
)

// ---------- frame builders (payloads verified on hardware) ----------

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

// viscaPanTiltAbsolute builds "absolute pan-tilt position":
// 81 01 06 02 VV WW <pan:4 nibbles> <tilt:4 nibbles> FF. pan/tilt are the
// camera's raw signed VISCA position units (two's-complement 16-bit).
func viscaPanTiltAbsolute(panSpeed, tiltSpeed byte, pan, tilt int16) []byte {
	out := []byte{viscaHeaderByte, 0x01, 0x06, 0x02, panSpeed, tiltSpeed}
	out = append(out, nibbles16(uint16(pan))...)
	out = append(out, nibbles16(uint16(tilt))...)
	return append(out, 0xFF)
}

// viscaZoomDirect builds "zoom direct": 81 01 04 47 <zoom:4 nibbles> FF.
func viscaZoomDirect(zoom uint16) []byte {
	out := []byte{viscaHeaderByte, 0x01, 0x04, 0x47}
	out = append(out, nibbles16(zoom)...)
	return append(out, 0xFF)
}

// viscaPanTiltInquiry builds "pan/tilt position inquiry": 81 09 06 12 FF.
// Reply: 90 50 <pan:4 nibbles> <tilt:4 nibbles> FF.
func viscaPanTiltInquiry() []byte { return []byte{viscaHeaderByte, 0x09, 0x06, 0x12, 0xFF} }

// viscaZoomInquiry builds "zoom position inquiry": 81 09 04 47 FF.
// Reply: 90 50 <zoom:4 nibbles> FF.
func viscaZoomInquiry() []byte { return []byte{viscaHeaderByte, 0x09, 0x04, 0x47, 0xFF} }

// viscaVersionInquiry builds "version inquiry": 81 09 00 02 FF (used as a health probe).
func viscaVersionInquiry() []byte { return []byte{viscaHeaderByte, 0x09, 0x00, 0x02, 0xFF} }

// ---------- nibble codecs (VISCA "expanded" 4-bit-per-byte encoding) ----------

// nibbles16 splits a 16-bit value into 4 bytes, each carrying one nibble in its
// low 4 bits, MSN first — VISCA's position/zoom wire form.
func nibbles16(v uint16) []byte {
	return []byte{byte(v>>12) & 0x0F, byte(v>>8) & 0x0F, byte(v>>4) & 0x0F, byte(v) & 0x0F}
}

// unnibble16 reassembles a 16-bit value from 4 low-nibble bytes.
func unnibble16(b []byte) uint16 {
	return uint16(b[0]&0x0F)<<12 | uint16(b[1]&0x0F)<<8 | uint16(b[2]&0x0F)<<4 | uint16(b[3]&0x0F)
}

// parsePanTiltReply extracts signed pan/tilt units from a position-inquiry reply
// (90 50 <pan4> <tilt4> FF, 11 bytes).
func parsePanTiltReply(frame []byte) (pan, tilt int16, err error) {
	if len(frame) != 11 || frame[0] != 0x90 || frame[1] != 0x50 {
		return 0, 0, fmt.Errorf("bad pan/tilt inquiry reply: % X", frame)
	}
	return int16(unnibble16(frame[2:6])), int16(unnibble16(frame[6:10])), nil
}

// parseZoomReply extracts the zoom value from a zoom-inquiry reply
// (90 50 <zoom4> FF, 7 bytes).
func parseZoomReply(frame []byte) (uint16, error) {
	if len(frame) != 7 || frame[0] != 0x90 || frame[1] != 0x50 {
		return 0, fmt.Errorf("bad zoom inquiry reply: % X", frame)
	}
	return unnibble16(frame[2:6]), nil
}

// ---------- transport: raw VISCA over TCP :5678 ----------

// viscaSend dials the device's VISCA TCP port, writes one raw VISCA frame, and
// returns the terminal reply frame. For commands it swallows the ACK (0x90 0x4y
// 0xFF) and returns the completion (0x90 0x5y 0xFF); for inquiries it returns the
// data completion (0x90 0x50 … 0xFF). A VISCA error reply (0x90 0x6y 0xFF) is
// surfaced as an error. host/creds come from the framework socketKey; VISCA on
// :5678 is unauthenticated, so credentials are ignored.
func viscaSend(socketKey string, frame []byte) ([]byte, error) {
	host, _, _ := parseSocketKey(socketKey)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h // strip the framework-appended HTTP port; VISCA uses viscaPort
	}
	addr := net.JoinHostPort(host, viscaPort)

	conn, err := net.DialTimeout("tcp", addr, viscaDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("visca dial %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(viscaDialTimeout))

	if _, err := conn.Write(frame); err != nil {
		return nil, fmt.Errorf("visca write: %w", err)
	}

	// Read 0xFF-delimited reply frames until a completion or error; skip ACKs.
	buf := make([]byte, 0, 32)
	one := make([]byte, 32)
	for {
		n, err := conn.Read(one)
		if n > 0 {
			buf = append(buf, one[:n]...)
			for {
				i := indexByte(buf, 0xFF)
				if i < 0 {
					break
				}
				reply := buf[:i+1]
				buf = buf[i+1:]
				if len(reply) < 2 || reply[0] != 0x90 {
					continue // not a camera-1 reply; ignore
				}
				switch reply[1] & 0xF0 {
				case 0x40: // ACK — command accepted, wait for completion
					continue
				case 0x50: // completion (with or without data)
					return reply, nil
				case 0x60: // error
					return nil, fmt.Errorf("visca error reply: % X", reply)
				default:
					continue
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("visca read (no completion): %w", err)
		}
	}
}

// indexByte returns the index of the first occurrence of b in s, or -1.
func indexByte(s []byte, b byte) int {
	for i, c := range s {
		if c == b {
			return i
		}
	}
	return -1
}
