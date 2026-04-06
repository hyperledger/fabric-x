#!/usr/bin/env bash
set -euo pipefail

HOST=${1:-localhost:9300}
URL="http://${HOST}/endorser/init"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS "$URL" \
  -H "Origin: http://evil.example" \
  -H "Access-Control-Request-Method: POST" || true)

if [ "$STATUS" == "403" ]; then
  echo "PASS: preflight rejected (403)"
  exit 0
else
  echo "FAIL: preflight returned $STATUS"
  exit 2
fi
