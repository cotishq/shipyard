#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8082}"
PROXY_URL="${PROXY_URL:-http://localhost:8001}"
API_KEY="${API_KEY:-shipyard_api_key_change_me_please}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-120}"
DEPLOY_TIMEOUT_SECONDS="${DEPLOY_TIMEOUT_SECONDS:-300}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-5}"

echo "Starting Shipyard stack..."
docker compose up --build -d

echo "Waiting for /healthz..."
deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))
while true; do
  if health_response="$(curl -fsS "${API_URL}/healthz")"; then
    echo "Health check response: ${health_response}"
    break
  fi

  if (( SECONDS >= deadline )); then
    echo "Timed out waiting for health endpoint"
    docker compose logs --tail=200 api worker postgres minio
    exit 1
  fi

  sleep "${POLL_INTERVAL_SECONDS}"
done

echo "Creating smoke deployment..."
deploy_response="$(curl -fsS -X POST "${API_URL}/deploy" \
  -H "X-API-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url":"https://github.com/mdn/beginner-html-site",
    "build_preset":"static-copy",
    "output_dir":""
  }')"

echo "Deployment response: ${deploy_response}"

deployment_id="$(printf '%s' "${deploy_response}" | grep -oE '"deployment_id":"[^"]+"' | cut -d'"' -f4)"
if [[ -z "${deployment_id}" ]]; then
  echo "Failed to extract deployment_id"
  exit 1
fi

echo "Polling deployment ${deployment_id}..."
deadline=$((SECONDS + DEPLOY_TIMEOUT_SECONDS))
final_status=""
while true; do
  deployment_json="$(curl -fsS "${API_URL}/deployments/${deployment_id}" -H "X-API-Key: ${API_KEY}")"
  echo "Deployment state: ${deployment_json}"

  final_status="$(printf '%s' "${deployment_json}" | grep -oE '"status":"[^"]+"' | cut -d'"' -f4)"
  if [[ "${final_status}" == "READY" ]]; then
    break
  fi

  if [[ "${final_status}" == "FAILED" ]]; then
    echo "Deployment failed"
    curl -fsS "${API_URL}/logs/${deployment_id}" -H "X-API-Key: ${API_KEY}" || true
    exit 1
  fi

  if (( SECONDS >= deadline )); then
    echo "Timed out waiting for deployment to finish"
    curl -fsS "${API_URL}/logs/${deployment_id}" -H "X-API-Key: ${API_KEY}" || true
    exit 1
  fi

  sleep "${POLL_INTERVAL_SECONDS}"
done

echo "Checking deployment logs..."
logs_json="$(curl -fsS "${API_URL}/logs/${deployment_id}" -H "X-API-Key: ${API_KEY}")"
echo "${logs_json}"
printf '%s' "${logs_json}" | grep -q 'Deployment ready'

echo "Checking served artifact via API..."
api_body="$(curl -fsS "${API_URL}/${deployment_id}")"
printf '%s' "${api_body}" | grep -q 'Mozilla is cool'

echo "Checking served artifact via NGINX..."
proxy_body="$(curl -fsS "${PROXY_URL}/${deployment_id}")"
printf '%s' "${proxy_body}" | grep -q 'Mozilla is cool'

echo "Smoke test passed for deployment ${deployment_id}"
