#!/bin/bash

# Set your environment variables before running
MICROSERVICE_URL="your.microservice.url"
DEVICE_FQDN="admin:password@your.ec20.ip"

echo "Running Epiphan EC20 Microservice Tests..."

# GET requests
echo ""
echo "=== GET status ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/status"
sleep 1

echo ""
echo "=== GET healthcheck ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/healthcheck"
sleep 1

echo ""
echo "=== GET ptzposition ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzposition"
sleep 1

echo ""
echo "=== GET presets ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/presets"
sleep 1

echo ""
echo "=== GET preview ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/preview"
sleep 1

# PUT requests
echo ""
echo "=== SET ptz (pan=45, tilt=-10, zoom=2.0) ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptz/45/-10" -H "Content-Type: application/json" -d '{"zoom":2.0}'
sleep 2

echo ""
echo "=== GET ptzposition (should reflect new position) ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzposition"
sleep 1

echo ""
echo "=== SET ptzhome ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzhome"
sleep 2

echo ""
echo "=== SET preset recall (preset 1) ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/preset/1"
sleep 2

echo ""
echo "=== SET presetsave (preset 1 as 'Center') ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/presetsave/1" -H "Content-Type: application/json" -d '"Center"'
sleep 1

echo ""
echo "=== SET tracking enable (presenter mode) ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/tracking/enable" -H "Content-Type: application/json" -d '"presenter"'
sleep 2

echo ""
echo "=== SET tracking disable ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/tracking/disable"
sleep 1

echo ""
echo "Tests complete."
