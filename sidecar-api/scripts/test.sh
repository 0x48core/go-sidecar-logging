#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8081"
PASS=0
FAIL=0

check() {
  local desc=$1
  local expected=$2
  local actual=$3
  if [ "$actual" = "$expected" ]; then
    echo "  PASS: $desc"
    PASS=$((PASS+1))
  else
    echo "  FAIL: $desc (expected=$expected, got=$actual)"
    FAIL=$((FAIL+1))
  fi
}

echo "=== sidecar-api integration tests ==="

# Test 1: GET /logs returns 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/logs")
check "GET /logs returns 200" "200" "$STATUS"

# Test 2: response has entries array
BODY=$(curl -s "$BASE_URL/logs")
HAS_ENTRIES=$(echo "$BODY" | grep -c '"entries"' || true)
check "response contains entries field" "1" "$HAS_ENTRIES"

# Test 3: count is a number
COUNT=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['count'])" 2>/dev/null || echo "parse_error")
if [ "$COUNT" != "parse_error" ]; then
  echo "  PASS: count field is present ($COUNT entries)"
  PASS=$((PASS+1))
else
  echo "  FAIL: could not parse count from response"
  FAIL=$((FAIL+1))
fi

echo ""
echo "=== results: $PASS passed, $FAIL failed ==="
[ $FAIL -eq 0 ]