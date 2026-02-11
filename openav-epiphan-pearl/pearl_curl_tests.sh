#!/bin/bash

# Set your environment variables before running
MICROSERVICE_URL="your.microservice.url"
DEVICE_FQDN="admin:password@your.pearl.ip"

echo "Running Epiphan Pearl Microservice Tests..."

# GET requests
echo ""
echo "=== GET status ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/status"
sleep 1

echo ""
echo "=== GET recordingstatus ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/recordingstatus"
sleep 1

echo ""
echo "=== GET storages ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/storages"
sleep 1

echo ""
echo "=== GET channels ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/channels"
sleep 1

echo ""
echo "=== GET healthcheck ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/healthcheck"
sleep 1

# PUT requests
echo ""
echo "=== SET recording start ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/recording" -H "Content-Type: application/json" -d '"start"'
sleep 5

echo ""
echo "=== GET recordingstatus (should be recording) ==="
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/recordingstatus"
sleep 1

echo ""
echo "=== SET recording stop ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/recording" -H "Content-Type: application/json" -d '"stop"'
sleep 1

echo ""
echo "=== SET singletouch start ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/singletouch" -H "Content-Type: application/json" -d '"start"'
sleep 5

echo ""
echo "=== SET singletouch stop ==="
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/singletouch" -H "Content-Type: application/json" -d '"stop"'
sleep 1

echo ""
echo "Tests complete."
