#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

WRANGLER_ENV="${WRANGLER_ENV:-}"
SKIP_NPM_INSTALL="${SKIP_NPM_INSTALL:-0}"

require_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}" >&2
    exit 1
  fi
}

wrangler() {
  npx --yes wrangler "$@"
}

put_secret() {
  local name="$1"
  local value="$2"
  local -a cmd=(secret put "${name}")

  if [[ -n "${WRANGLER_ENV}" ]]; then
    cmd+=(--env "${WRANGLER_ENV}")
  fi

  printf '%s' "${value}" | wrangler "${cmd[@]}" >/dev/null
  echo "Set Cloudflare secret: ${name}"
}

require_cmd npm
require_cmd docker

if [[ "${SKIP_NPM_INSTALL}" != "1" ]]; then
  npm install
fi

if ! wrangler whoami >/dev/null 2>&1; then
  echo "Wrangler is not authenticated. Run: npx wrangler login" >&2
  exit 1
fi

if [[ -z "${MISTER_MORPH_LLM_API_KEY:-}" ]]; then
  echo "MISTER_MORPH_LLM_API_KEY is required." >&2
  echo "Example: MISTER_MORPH_LLM_API_KEY=... ./deploy.sh" >&2
  exit 1
fi

GENERATED_SERVER_TOKEN=0
if [[ -z "${MISTER_MORPH_SERVER_AUTH_TOKEN:-}" ]]; then
  if command -v openssl >/dev/null 2>&1; then
    MISTER_MORPH_SERVER_AUTH_TOKEN="$(openssl rand -hex 24)"
  else
    MISTER_MORPH_SERVER_AUTH_TOKEN="$(date +%s%N | sha256sum | cut -c1-48)"
  fi
  GENERATED_SERVER_TOKEN=1
fi

put_secret "MISTER_MORPH_LLM_API_KEY" "${MISTER_MORPH_LLM_API_KEY}"
put_secret "MISTER_MORPH_SERVER_AUTH_TOKEN" "${MISTER_MORPH_SERVER_AUTH_TOKEN}"

if [[ -n "${MISTER_MORPH_TELEGRAM_BOT_TOKEN:-}" ]]; then
  put_secret "MISTER_MORPH_TELEGRAM_BOT_TOKEN" "${MISTER_MORPH_TELEGRAM_BOT_TOKEN}"
fi

echo "Deploying Cloudflare Worker + Container..."
if [[ -n "${WRANGLER_ENV}" ]]; then
  wrangler deploy --env "${WRANGLER_ENV}" "$@"
else
  wrangler deploy "$@"
fi

echo
if [[ "${GENERATED_SERVER_TOKEN}" == "1" ]]; then
  echo "Generated MISTER_MORPH_SERVER_AUTH_TOKEN:"
  echo "${MISTER_MORPH_SERVER_AUTH_TOKEN}"
  echo
fi

echo "Example health check (replace worker domain):"
echo "curl -H \"Authorization: Bearer ${MISTER_MORPH_SERVER_AUTH_TOKEN}\" https://<worker-domain>/health"
