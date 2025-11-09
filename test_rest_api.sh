#!/bin/bash

set -e

API_URL="http://localhost:8080/api/v1"

echo "==================================="
echo "Centro REST API Test Script"
echo "==================================="
echo ""

echo "1. Login to get JWT token..."
TOKEN_RESPONSE=$(curl -s -X POST ${API_URL}/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo $TOKEN_RESPONSE | jq -r '.token')

if [ "$TOKEN" == "null" ]; then
  echo "❌ Failed to get token"
  echo "Response: $TOKEN_RESPONSE"
  exit 1
fi

echo "✅ Token obtained: ${TOKEN:0:20}..."
echo ""

echo "2. Get system statistics..."
curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/stats | jq '.'
echo ""

echo "3. List all nodes..."
curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/nodes | jq '.'
echo ""

echo "4. List all jobs..."
curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/jobs | jq '.'
echo ""

echo "5. Submit a new job..."
JOB_RESPONSE=$(curl -s -X POST ${API_URL}/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-nginx",
    "type": "service",
    "datacenters": "dc1",
    "tasks": [
      {
        "name": "web",
        "driver": "podman",
        "config": {
          "image": "nginx:latest",
          "options": {
            "port": "8081:80"
          }
        },
        "env": {
          "NODE_ENV": "production"
        }
      }
    ],
    "meta": {
      "created_by": "test_script",
      "priority": "normal"
    }
  }')

echo $JOB_RESPONSE | jq '.'
JOB_ID=$(echo $JOB_RESPONSE | jq -r '.job_id')
echo ""

if [ "$JOB_ID" != "null" ]; then
  echo "6. Get job details (Job ID: $JOB_ID)..."
  sleep 2
  curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/jobs/${JOB_ID} | jq '.'
  echo ""
  
  echo "7. Get job status..."
  curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/jobs/${JOB_ID}/status | jq '.'
  echo ""
  
  echo "8. Get job events..."
  curl -s -H "Authorization: Bearer $TOKEN" ${API_URL}/jobs/${JOB_ID}/events | jq '.'
  echo ""
fi

echo "==================================="
echo "Test completed successfully!"
echo "==================================="

