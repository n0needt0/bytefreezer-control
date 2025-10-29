#!/bin/bash

  # Test with system_admin
  echo "=== Testing with system_admin ==="
  SYSTEM_TOKEN=$(curl -s -X POST http://192.168.86.103:8082/api/v1/login \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@example.com","password":"your_password"}' \
    | jq -r '.token')

  echo "System admin token: ${SYSTEM_TOKEN:0:20}..."

  curl -s -X GET "http://192.168.86.103:8082/api/v1/errors?status=active&limit=5" \
    -H "Authorization: Bearer $SYSTEM_TOKEN" \
    | jq '{count: .count, total: .total, sample_errors: .errors[0:3] | map({id, account_id, component})}'

  echo ""
  echo "=== Testing with account_admin ==="
  ACCOUNT_TOKEN=$(curl -s -X POST http://192.168.86.103:8082/api/v1/login \
    -H "Content-Type: application/json" \
    -d '{"email":"account_admin@example.com","password":"your_password"}' \
    | jq -r '.token')

  echo "Account admin token: ${ACCOUNT_TOKEN:0:20}..."

  curl -s -X GET "http://192.168.86.103:8082/api/v1/errors?status=active&limit=5" \
    -H "Authorization: Bearer $ACCOUNT_TOKEN" \
    | jq '{count: .count, total: .total, sample_errors: .errors[0:3] | map({id, account_id, component})}'

  echo ""
  echo "=== Checking middleware logs ==="
  ssh andrew@192.168.86.103 "tail -100 /tmp/control-debug.log | grep -E 'JWT MIDDLEWARE|ACCESS CONTROL'"
