#!/bin/bash
# Smart Room Demo — Stack Verification
# Usage: ./test-stack.sh
set -e

echo "=== Smart Room Demo Stack Test ==="

echo "1. Checking containers..."
docker compose ps --format "table {{.Name}}\t{{.Status}}"

echo ""
echo "2. Testing orchestrator..."
ORCH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ 2>/dev/null || echo "000")
if [ "$ORCH_STATUS" = "000" ]; then
  echo "   FAIL: orchestrator unreachable"
else
  echo "   OK: orchestrator responded with HTTP $ORCH_STATUS"
fi

echo ""
echo "3. Testing Pearl microservice (via orchestrator network)..."
PEARL_STATUS=$(docker compose exec -T orchestrator curl -sf http://microservice-epiphan-pearl:80/ 2>/dev/null && echo "OK" || echo "FAIL: not responding")
echo "   $PEARL_STATUS"

echo ""
echo "4. Testing EC20 microservice (via orchestrator network)..."
EC20_STATUS=$(docker compose exec -T orchestrator curl -sf http://microservice-epiphan-ec20:80/ 2>/dev/null && echo "OK" || echo "FAIL: not responding")
echo "   $EC20_STATUS"

echo ""
echo "=== Done ==="
