# Nak broadcast tests (for agents)

Canonical procedure: also listed from [AGENTS.md](../AGENTS.md). Humans: [README.md](../README.md) Testing section.

Reproducible checks that Espelho de São Miguel (1) fans events out to local
websocket subscribers without going through khatru `BroadcastEvent`, and (2)
publishes those events to other relays.

Do **not** invent results. Run the script (or the equivalent commands below)
against a live relay and report the script output.

## Prerequisites

- Live relay HTTP + websocket (compose default: `http://127.0.0.1:3337` and `ws://127.0.0.1:3337`)
- `nak` on `PATH` (`go install github.com/fiatjaf/nak@latest`)
- `curl`, `python3`, `timeout` (GNU coreutils)
- Health endpoint returning JSON (`GET /api/v1/health`)

If the container was just started, wait until HTTP is up. Broadcast discovery
can take several minutes; `/api/v1/live` then `/api/v1/health` must succeed
before these tests.

## One-shot script

From the repo root:

```bash
./scripts/nak-broadcast-tests.sh
```

Overrides:

| Variable | Default | Meaning |
|---|---|---|
| `RELAY` | `ws://127.0.0.1:3337` | Local websocket |
| `STATS_URL` | `http://127.0.0.1:3337/api/v1/stats` | JSON stats (top relays) |
| `HEALTH_URL` | `http://127.0.0.1:3337/api/v1/health` | Readiness |
| `MANDATORY_RELAYS` | `wss://nostr.girino.org,wss://relay.primal.net,wss://nos.lol,wss://wot.girino.org` | Comma-separated dest relays to query |
| `OUTBOUND_WAIT` | `12` | Seconds to wait after publish before querying remotes |
| `NON_MANDATORY_COUNT` | `3` | How many non-mandatory top relays to confirm |
| `NAK` | `nak` | Binary path |
| `WORKDIR` | temp dir | Where logs are written |

Exit codes:

- `0` — local fan-out delivered both events to all 3 readers **and** at least one mandatory relay has the outbound event
- `1` — local fan-out failed, publish failed, health never came up, or no mandatory hit

A `WARN` about fewer than `NON_MANDATORY_COUNT` non-mandatory hits is **not** a
script failure. Many top-N relays require AUTH, return 429, or drop unknown
pubkeys. Record which URLs hit.

## What each phase proves

### Phase 1 — local fan-out

Three `nak req --stream` clients subscribe with a unique `#t` tag. Two `nak event`
processes publish in parallel with that tag.

Pass = each reader JSONL contains both contents (`…-A` and `…-B`).

This path is `OnEventSaved` → `fanout.Hub.BroadcastEvent` → per-websocket write queue,
**not** khatru `notifyListeners`/`BroadcastEvent` (`PreventBroadcast` skips those
sockets). If this phase fails, local isolation is broken.

### Phase 2 — outbound broadcast

One new event is published to the local relay. After `OUTBOUND_WAIT` seconds the
script queries:

1. `MANDATORY_RELAYS` with `nak req --id <event-id>`
2. `manager.top_relays` from `/api/v1/stats`, skipping mandatory URLs, until
   `NON_MANDATORY_COUNT` hits (or the list ends)

`broadcaststore` success count going up by 1 only means the event was queued
locally. **Presence on a remote via `nak req --id` is the only proof it arrived.**

Known non-failures when a dest misses:

- `wss://nos.lol` — HTTP 429 from this host is common
- `wss://wot.girino.org` — WoT often rejects a freshly generated pubkey
- `wss://haven.girino.org/*` — AUTH / owner-only
- path-style `nostr1.com/…` relays in top-N — AUTH or not a public kind:1 store

## Manual equivalent (if the script cannot run)

Local fan-out:

```bash
TAG="smm-fanout-$(date +%s)"
RELAY=ws://127.0.0.1:3337
# terminal 1–3:
nak req --stream -k 1 -t t=$TAG --since now $RELAY
# after all three print "connecting … ok", two parallel publishes:
nak event --sec "$(nak key generate)" -k 1 -t t=$TAG -c "$TAG-A" $RELAY
nak event --sec "$(nak key generate)" -k 1 -t t=$TAG -c "$TAG-B" $RELAY
# each reader must print both events
```

Outbound:

```bash
SEC=$(nak key generate)
nak event --sec "$SEC" -k 1 -c "outbound-$(date +%s)" $RELAY | tee /tmp/ev.json
ID=$(python3 -c 'import json; print(json.load(open("/tmp/ev.json"))["id"])')
sleep 12
nak req --id "$ID" -l 3 wss://nostr.girino.org
nak req --id "$ID" -l 3 wss://relay.primal.net
# plus any non-mandatory URL from GET /api/v1/stats → manager.top_relays
```

## After compose rebuild

```bash
docker compose up -d --build relay
# wait until curl -fsS http://127.0.0.1:3337/api/v1/health succeeds
./scripts/nak-broadcast-tests.sh
```
