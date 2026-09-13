#!/usr/bin/env bash
# Reproducible nak checks for Espelho de São Miguel.
# Verifies (1) local async fan-out and (2) outbound broadcast to other relays.
#
# Usage:
#   ./scripts/nak-broadcast-tests.sh
#   RELAY=ws://127.0.0.1:3337 STATS_URL=http://127.0.0.1:3337/api/v1/stats ./scripts/nak-broadcast-tests.sh
#
# Requires: nak, curl, python3
# Exit 0 only if local fan-out PASS and at least one outbound relay has the event.
set -u

RELAY="${RELAY:-ws://127.0.0.1:3337}"
STATS_URL="${STATS_URL:-http://127.0.0.1:3337/api/v1/stats}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:3337/api/v1/health}"
NAK="${NAK:-nak}"
OUTBOUND_WAIT="${OUTBOUND_WAIT:-12}"
NON_MANDATORY_COUNT="${NON_MANDATORY_COUNT:-3}"
WORKDIR="${WORKDIR:-$(mktemp -d /tmp/nak-smm-XXXX)}"

# Default mandatory set used in production; override with a comma-separated list.
MANDATORY_RELAYS="${MANDATORY_RELAYS:-wss://nostr.girino.org,wss://relay.primal.net,wss://nos.lol,wss://wot.girino.org}"

log() { printf '%s\n' "$*"; }
die() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

command -v "$NAK" >/dev/null || die "nak not found (install: go install github.com/fiatjaf/nak@latest)"
command -v curl >/dev/null || die "curl not found"
command -v python3 >/dev/null || die "python3 not found"

log "workdir=$WORKDIR"
log "relay=$RELAY"

# --- wait for HTTP ---
log "== waiting for $HEALTH_URL =="
ready=0
for i in $(seq 1 60); do
  if curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
[ "$ready" = 1 ] || die "health endpoint never became ready"
log "health ok"

# --- phase 1: local fan-out (3 readers, 2 writers, unique tag) ---
log "== phase 1: local fan-out =="
TAG="smm-fanout-$(date +%s)-$$"
MARK="fanout-probe-${TAG}"
SEC1=$("$NAK" key generate)
SEC2=$("$NAK" key generate)

for i in 1 2 3; do
  timeout 35 "$NAK" req --stream -k 1 -t "t=${TAG}" --since now "$RELAY" \
    >"$WORKDIR/r${i}.jsonl" 2>"$WORKDIR/r${i}.err" &
  echo $! >"$WORKDIR/r${i}.pid"
done

connected=0
for n in $(seq 1 15); do
  connected=$(grep -l 'connecting to' "$WORKDIR"/r*.err 2>/dev/null | wc -l)
  [ "$connected" -ge 3 ] && break
  sleep 1
done
sleep 2

timeout 20 "$NAK" event --sec "$SEC1" -k 1 -t "t=${TAG}" -c "${MARK}-A" "$RELAY" \
  >"$WORKDIR/w1.out" 2>"$WORKDIR/w1.err" &
W1=$!
timeout 20 "$NAK" event --sec "$SEC2" -k 1 -t "t=${TAG}" -c "${MARK}-B" "$RELAY" \
  >"$WORKDIR/w2.out" 2>"$WORKDIR/w2.err" &
W2=$!
wait "$W1"; e1=$?
wait "$W2"; e2=$?
[ "$e1" = 0 ] && [ "$e2" = 0 ] || die "local publish failed (w1=$e1 w2=$e2)"
grep -q 'success' "$WORKDIR/w1.err" && grep -q 'success' "$WORKDIR/w2.err" \
  || die "local publish did not report success"

sleep 4

python3 - "$WORKDIR" "$MARK" <<'PY' || die "local fan-out did not deliver both events to all readers"
import json, os, sys
d, mark = sys.argv[1], sys.argv[2]
want = {mark + "-A", mark + "-B"}
ok = True
for i in (1, 2, 3):
    path = os.path.join(d, f"r{i}.jsonl")
    got = set()
    if os.path.exists(path):
        for line in open(path):
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                continue
            c = ev.get("content", "")
            if mark in c:
                got.add(c)
    missing = want - got
    status = "OK" if not missing else "MISSING " + str(missing)
    print(f"  reader{i}: {status} got={got}")
    if missing:
        ok = False
if not ok:
    sys.exit(1)
print("  local fan-out PASS")
PY

for i in 1 2 3; do
  if [ -f "$WORKDIR/r${i}.pid" ]; then
    kill "$(cat "$WORKDIR/r${i}.pid")" 2>/dev/null || true
  fi
done
sleep 1

# --- phase 2: outbound broadcast ---
log "== phase 2: outbound broadcast =="
OTAG="smm-out-$(date +%s)"
OSEC=$("$NAK" key generate)
OCONTENT="outbound-probe-${OTAG}"
timeout 20 "$NAK" event --sec "$OSEC" -k 1 -t "t=${OTAG}" -c "$OCONTENT" "$RELAY" \
  >"$WORKDIR/out.event" 2>"$WORKDIR/out.err" || die "outbound publish failed"
OID=$(python3 -c 'import json; print(json.load(open("'"$WORKDIR"'/out.event"))["id"])')
log "  event_id=$OID"
grep -q 'success' "$WORKDIR/out.err" || die "outbound publish did not report success"
log "  waiting ${OUTBOUND_WAIT}s for workers..."
sleep "$OUTBOUND_WAIT"

IFS=',' read -r -a MANDATORY <<< "$MANDATORY_RELAYS"

query_id() {
  local url="$1"
  local tries="${2:-2}"
  local i out err
  for i in $(seq 1 "$tries"); do
    out=$(mktemp "$WORKDIR/q.XXXX")
    err=$(mktemp "$WORKDIR/e.XXXX")
    if timeout 12 "$NAK" req --id "$OID" -l 3 "$url" >"$out" 2>"$err"; then
      :
    fi
    if grep -q "$OID" "$out" 2>/dev/null; then
      echo "FOUND"
      rm -f "$out" "$err"
      return 0
    fi
    rm -f "$out" "$err"
    sleep 2
  done
  echo "MISS"
  return 1
}

mandatory_found=0
log "  mandatory:"
for r in "${MANDATORY[@]}"; do
  r=$(echo "$r" | tr -d ' ')
  [ -n "$r" ] || continue
  if res=$(query_id "$r" 3); then
    log "    FOUND  $r"
    mandatory_found=$((mandatory_found + 1))
  else
    log "    MISS   $r"
  fi
done

# Non-mandatory: take top_relays from /stats, skip mandatory URLs, probe until N found or list exhausted.
curl -fsS --max-time 8 "$STATS_URL" >"$WORKDIR/stats.json" || die "failed to fetch $STATS_URL"
mapfile -t CANDS < <(python3 - "$WORKDIR/stats.json" "$MANDATORY_RELAYS" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
mand = {x.strip().rstrip("/") for x in sys.argv[2].split(",") if x.strip()}
seen = set()
for item in (d.get("manager") or {}).get("top_relays") or []:
    u = (item.get("url") or "").rstrip("/")
    if not u or u in mand or u in seen:
        continue
    seen.add(u)
    print(u)
PY
)

non_found=0
non_found_urls=()
log "  non-mandatory (from manager.top_relays):"
for r in "${CANDS[@]}"; do
  [ "$non_found" -ge "$NON_MANDATORY_COUNT" ] && break
  if query_id "$r" 1 >/dev/null; then
    log "    FOUND  $r"
    non_found=$((non_found + 1))
    non_found_urls+=("$r")
  else
    log "    miss   $r"
  fi
done

log "== summary =="
log "  local fan-out: PASS"
log "  mandatory hits: $mandatory_found / ${#MANDATORY[@]}"
log "  non-mandatory hits: $non_found (wanted $NON_MANDATORY_COUNT)"
for u in "${non_found_urls[@]+"${non_found_urls[@]}"}"; do
  log "    $u"
done
log "  event_id=$OID workdir=$WORKDIR"

if [ "$mandatory_found" -lt 1 ]; then
  die "no mandatory relay received the event"
fi
if [ "$non_found" -lt "$NON_MANDATORY_COUNT" ]; then
  log "WARN: fewer than $NON_MANDATORY_COUNT non-mandatory hits (rate-limits / AUTH / policy are common)"
fi
log "PASS"
exit 0
