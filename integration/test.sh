#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# Auto-detect architecture if not set
if [ -z "${GOARCH:-}" ]; then
    case "$(uname -m)" in
        x86_64|amd64) export GOARCH=amd64 ;;
        arm64|aarch64) export GOARCH=arm64 ;;
        *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
fi

COMPOSE="docker compose -f docker-compose.yaml"
MINISTACK_URL="http://localhost:4566"
GATEWAY_URL="http://localhost:8090"
FUNCTION_NAME="test-function"

cleanup() {
    echo "Cleaning up..."
    ${COMPOSE} down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "==> Starting services..."
${COMPOSE} up -d --build --wait

echo "==> Waiting for gateway to be ready..."
for i in $(seq 1 30); do
    if curl -sf "${GATEWAY_URL}/system/status" >/dev/null 2>&1; then
        echo "Gateway is ready."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: Gateway did not become ready in time."
        ${COMPOSE} logs gateway
        exit 1
    fi
    sleep 1
done

echo "==> Creating Lambda function in ministack..."
cd lambda
zip -qj /tmp/lambda-test.zip index.mjs
cd "${SCRIPT_DIR}"

AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
aws --endpoint-url "${MINISTACK_URL}" \
    --region us-east-1 \
    --no-cli-pager \
    lambda create-function \
    --function-name "${FUNCTION_NAME}" \
    --runtime nodejs20.x \
    --role "arn:aws:iam::000000000000:role/fake-role" \
    --handler index.handler \
    --zip-file fileb:///tmp/lambda-test.zip

echo "==> Testing gateway proxies request to Lambda..."
RESPONSE=$(curl -sf "${GATEWAY_URL}/${FUNCTION_NAME}/test/path")

echo "Response: ${RESPONSE}"

# Verify the response contains expected fields
if echo "${RESPONSE}" | grep -q '"message":"hello from lambda"'; then
    echo "PASS: Response body matches expected Lambda output."
else
    echo "FAIL: Unexpected response body."
    ${COMPOSE} logs gateway
    exit 1
fi

if echo "${RESPONSE}" | grep -q '"path":"/test/path"'; then
    echo "PASS: Path was correctly forwarded."
else
    echo "FAIL: Path was not correctly forwarded."
    ${COMPOSE} logs gateway
    exit 1
fi

if echo "${RESPONSE}" | grep -q '"method":"GET"'; then
    echo "PASS: HTTP method was correctly forwarded."
else
    echo "FAIL: HTTP method was not correctly forwarded."
    ${COMPOSE} logs gateway
    exit 1
fi

echo ""
echo "==> All integration tests passed."
