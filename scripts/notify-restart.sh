#!/bin/sh
# Post-restart hook for willfarrell/autoheal.
# Args: CONTAINER_NAME CONTAINER_SHORT_ID CONTAINER_STATE RESTART_TIMEOUT
# Runs inside the autoheal container after a restart attempt.
set -eu

CONTAINER_NAME="${1:-unknown}"
CONTAINER_SHORT_ID="${2:-}"
CONTAINER_STATE="${3:-unknown}"
RESTART_TIMEOUT="${4:-}"
DOCKER_SOCK="${DOCKER_SOCK:-/var/run/docker.sock}"
WEBHOOK_URL="${WEBHOOK_URL:-}"
CURL_TIMEOUT="${CURL_TIMEOUT:-30}"

log() {
  echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') notify-restart: $*"
}

docker_api() {
  path="$1"
  curl -sS --max-time "$CURL_TIMEOUT" --unix-socket "$DOCKER_SOCK" "http://localhost${path}" || true
}

# Prefer the short id; fall back to the name without a leading slash.
LOOKUP_ID="$CONTAINER_SHORT_ID"
if [ -z "$LOOKUP_ID" ] || [ "$LOOKUP_ID" = "null" ]; then
  LOOKUP_ID="${CONTAINER_NAME#/}"
fi

INSPECT_JSON="$(docker_api "/containers/${LOOKUP_ID}/json")"

HEALTH_SUMMARY="(no health log)"
if [ -n "$INSPECT_JSON" ] && command -v jq >/dev/null 2>&1; then
  HEALTH_SUMMARY="$(printf '%s' "$INSPECT_JSON" | jq -r '
    (.State.Health.Log // [])
    | if length == 0 then "(empty after restart — see container logs)"
      else
        .[-3:]
        | map(
            (.Start // "?")
            + " exit=" + ((.ExitCode // 0)|tostring)
            + "\n" + ((.Output // "") | gsub("\r"; "") | split("\n") | map(select(test("\\S"))) | .[-20:] | join("\n"))
          )
        | join("\n---\n")
      end
  ' 2>/dev/null || echo "(failed to parse health log)")"
fi

# Docker logs API is a multiplexed stream. timestamps=1 lets us grep ISO lines.
RAW_LOGS="$(curl -sS --max-time "$CURL_TIMEOUT" --unix-socket "$DOCKER_SOCK" \
  "http://localhost/containers/${LOOKUP_ID}/logs?stdout=1&stderr=1&timestamps=1&tail=120" || true)"

# Keep health/error/timeout lines; skip noisy whitelist rejects.
# Fall back to the last printable lines if nothing matches yet.
FILTERED_LOGS="$(printf '%s' "$RAW_LOGS" | tr -cd '\11\12\15\40-\176' | \
  grep -E 'health check:|health heartbeat:|health stats timed out|GetAllStats timed out|stats endpoint|\[ERROR\]|\[FATAL\]|unhealthy|stats_timeout' | tail -n 40 || true)"
if [ -z "$FILTERED_LOGS" ]; then
  FILTERED_LOGS="$(printf '%s' "$RAW_LOGS" | tr -cd '\11\12\15\40-\176' | \
    grep -E '\[ERROR\]|\[WARN\]|\[FATAL\]' | grep -v whitelist | tail -n 25 || true)"
fi
if [ -z "$FILTERED_LOGS" ]; then
  FILTERED_LOGS="$(printf '%s' "$RAW_LOGS" | tr -cd '\11\12\15\40-\176' | grep -E '20[0-9]{2}-' | tail -n 15 || true)"
fi
if [ -z "$FILTERED_LOGS" ]; then
  FILTERED_LOGS="(no container logs retrieved)"
fi

NAME_DISPLAY="${CONTAINER_NAME#/}"
HOST="$(hostname 2>/dev/null || echo unknown)"
NOW="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"

MSG="$(cat <<EOF
Autoheal restarted **${NAME_DISPLAY}** (\`${CONTAINER_SHORT_ID}\`)
host=${HOST}  state_before=${CONTAINER_STATE}  stop_timeout=${RESTART_TIMEOUT}s  at=${NOW}

**Last Docker healthcheck output**
${HEALTH_SUMMARY}

**Recent container logs (health/error)**
${FILTERED_LOGS}
EOF
)"

# Discord content is capped at 2000 characters.
if [ "${#MSG}" -gt 1900 ]; then
  MSG="$(printf '%s' "$MSG" | head -c 1900)
…(truncated)"
fi

log "container=${NAME_DISPLAY} id=${CONTAINER_SHORT_ID} state=${CONTAINER_STATE}"
log "health summary:"
printf '%s\n' "$HEALTH_SUMMARY"
log "filtered logs:"
printf '%s\n' "$FILTERED_LOGS"

if [ -z "$WEBHOOK_URL" ]; then
  log "WEBHOOK_URL empty; skipping Discord post"
  exit 0
fi

if command -v jq >/dev/null 2>&1; then
  PAYLOAD="$(jq -n --arg content "$MSG" '{content: $content}')"
else
  ESCAPED="$(printf '%s' "$MSG" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')"
  PAYLOAD="{\"content\":\"${ESCAPED}\"}"
fi

curl -sS --max-time "$CURL_TIMEOUT" -X POST -H 'Content-Type: application/json' \
  -d "$PAYLOAD" "$WEBHOOK_URL" >/dev/null || log "webhook post failed"
log "webhook posted"
