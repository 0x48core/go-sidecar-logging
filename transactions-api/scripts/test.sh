#!/usr/bin/env bash
set -e

BASE_URL="http://localhost:8080"
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

echo "=== transactions-api integration tests ==="

# Test 1: valid transaction
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/transactions" \
  -H "Content-Type: application/json" \
  -d '{"id":"tx-001","amount":99.50,"from":"alice","to":"bob"}')
check "valid transaction returns 202" "202" "$STATUS"

# Test 2: missing fields
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/transactions" \
  -H "Content-Type: application/json" \
  -d '{"id":"tx-002"}')
check "missing fields returns 400" "400" "$STATUS"

# Test 3: negative amount
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/transactions" \
  -H "Content-Type: application/json" \
  -d '{"id":"tx-003","amount":-5,"from":"alice","to":"bob"}')
check "negative amount returns 400" "400" "$STATUS"

# Test 4: send 5 more to trigger flush
echo ""
echo "--- sending 5 transactions to trigger flush ---"
for i in {2..6}; do
  curl -s -o /dev/null -X POST "$BASE_URL/transactions" \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"tx-00$i\",\"amount\":$((i*10)).0,\"from\":\"alice\",\"to\":\"bob\"}"
done

echo "--- waiting 6s for flush interval ---"
sleep 6

# Test 5: verify log file written
LOG_FILE="./logs/xapi.log"
if [ -f "$LOG_FILE" ]; then
  COUNT=$(grep -c "INFO" "$LOG_FILE")
  echo "  PASS: log file exists with $COUNT entries"
  PASS=$((PASS+1))
else
  echo "  FAIL: log file not found at $LOG_FILE"
  FAIL=$((FAIL+1))
fi

echo ""
echo "=== results: $PASS passed, $FAIL failed ==="
[ $FAIL -eq 0 ]