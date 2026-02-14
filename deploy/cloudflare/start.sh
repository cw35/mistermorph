#!/usr/bin/env sh
set -eu

MODE="${MISTER_MORPH_RUN_MODE:-serve}"
LOG_LEVEL="${MISTER_MORPH_LOG_LEVEL:-info}"
CONFIG_PATH="${MISTER_MORPH_CONFIG_FILE:-/app/config.yaml}"

echo "mistermorph_boot mode=${MODE} log_level=${LOG_LEVEL} config=${CONFIG_PATH}" >&2

case "${MODE}" in
  serve)
    exec /usr/local/bin/mistermorph --config "${CONFIG_PATH}" serve --log-level "${LOG_LEVEL}"
    ;;
  telegram)
    exec /usr/local/bin/mistermorph --config "${CONFIG_PATH}" telegram --log-level "${LOG_LEVEL}"
    ;;
  *)
    echo "Unsupported MISTER_MORPH_RUN_MODE: ${MODE} (expected: serve|telegram)" >&2
    exit 1
    ;;
esac
