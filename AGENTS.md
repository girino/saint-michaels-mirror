# Agent notes — Espelho de São Miguel

Facts for coding agents. Humans: start at [README.md](README.md).

## Repo

Go Nostr relay aggregator (`cmd/saint-michaels-mirror`). Compose template is `docker-compose.prod.yml`. Local `docker-compose.yml` and `.env` are gitignored — do not commit them, `tor-exits.txt`, or webhook/nsec values.

Default local websocket/HTTP: `ws://127.0.0.1:3337` / `http://127.0.0.1:3337`.

## HTTP vs websocket

| Path | Role |
|---|---|
| `GET /api/v1/live` | Process-up only. No stats. Use this to wait for listen. |
| `GET /api/v1/health` | Subsystem colors. HTTP **503** if any is RED (Docker `curl -f` → unhealthy → autoheal). Skips typed-nil stats providers (CI has no `BROADCAST_SEED_RELAYS`, so `*BroadcastStore` is nil). |
| `GET /api/v1/stats` | Full collector, 5s timeout. Includes `manager.GetStats()` which **deadlocks** if it `RLock`s then calls `GetTopRelays()` (also `RLock`) while a writer waits. Do not add health probes that call `GetAllStats()`. |

HTTP **does not listen** until query-remote/broadcast init finishes (often 2–5 min with thousands of relays). Docker publishes the port immediately → early `curl` is `Empty reply` / `Connection reset`. Compose `start_period` is **600s**. CI waits on `/live` then `/health` (`.github/workflows/test.yml`).

Do not use khatru `GetListeningFilters()` — it races (`index out of range`) when REQs mutate `listeners`. Listener count is `clientHub` / `subTracker` (allowed REQ increment, disconnect decrement).

## Mirror and writes

Do **not** send live events through `khatru.Relay.BroadcastEvent` / `notifyListeners`. Sequential `WriteJSON` with no deadline: one slow socket blocks the firehose and go-nostr `dispatchEvent` leaks goroutines (YELLOW 30k / RED 100k → 503 → autoheal).

Current path:

- Ingest from `QUERY_REMOTES` only while some client has an open REQ (`dropping_mirror.go`). Queue 4096; drop with rate-limited WARN if full.
- Fan-out via `clientHub` (`client_fanout.go`): per-websocket queue (64) + writer goroutine. Slow client drops only its own events (`client_skips`). Write >200ms logged; stuck >3s disconnects **that** socket.
- `PreventBroadcast` skips khatru sync writes for sockets we already write. Client `EVENT` also fans out from `OnEventSaved`.
- Max **256** concurrent websockets (`connections.go`).
- Whitelist (`ALLOWED_NPUBS`): AUTH on connect; non-member or 8s timeout **closes** the websocket.

Proof an event reached a **remote** relay is `nak req --id <id>` on that URL, not `broadcaststore.successes++` (that only means queued locally).

## Autoheal

`willfarrell/autoheal` + `WEBHOOK_URL`. After restart, `scripts/notify-restart.sh` posts last Docker health output + recent health/ERROR logs. Mount is in compose. Short autoheal message still fires first.

## Tests

```bash
go test ./cmd/saint-michaels-mirror/
./scripts/nak-broadcast-tests.sh
```

Nak procedure, env vars, pass/fail, known dest misses: [doc/NAK_BROADCAST_TESTS.md](doc/NAK_BROADCAST_TESTS.md). Run it; do not invent results. `nos.lol` 429, WoT rejecting a fresh pubkey, and haven AUTH are expected misses, not proof broadcast is broken.

## Pitfalls

- Typed-nil `*T` inside a `statsProvider` interface is not `p == nil`; `GetStats()` panics and HTTP closes the conn (`curl 52`). `isNilStatsProvider` handles this.
- Panic in a `go func() { ch <- p.GetStats() }` kills the **process** unless recovered (health path recovers).
- Do not log every dropped mirror event (resets a “first drop” counter and livelocks ingest).
- `VERBOSE=mirror.StartMirroring` is stale; mirroring is `dropping_mirror` / `client_fanout`, not nostr-lib `MirrorManager.StartMirroring`.
