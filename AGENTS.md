# Agent instructions — Espelho de São Miguel

This file is for AI agents working in this repository. Keep changes focused,
verify before claiming success, and never mask an operational problem with a
workaround in place of a diagnosis.

## Project

Nostr relay aggregator built on [khatru](https://github.com/fiatjaf/khatru).
Go service, shipped as a container image
(`ghcr.io/girino/saint-michaels-mirror/saint-michaels-mirror`). Production runs
on a small VPS (3 vCPU, ~3.4 GB RAM) behind a Tor frontend.

## Repository state vs production (read before trusting `main`)

`main` **does not contain** the previous goroutine-leak intervention. Those
changes live on the unmerged branch `test/nostr-lib-fanout` (also reachable from
`fix/health-unhealthy-diagnostics`):

- `4788bcb` — "Fix unhealthy restarts from goroutine leaks and slow websocket
  writes" (adds `health_api.go`, dropping mirror, bounded per-socket write
  queues, connection cap, AUTH timeout, goroutine dump);
- `1fbd93b` — switches fanout/mirror to `github.com/girino/nostr-lib` and bumps
  it to `v0.0.0-20260913230849-1c87b9904f51` (`main` still pins the older
  `v0.0.0-20251027142055-a7108048b09e`);
- `6b79991`, `edde375` — goroutine-dump control, log rotation, klauspost CVE.

**Production is running that branch, not `main`.** The deployed binary embeds
`nostr-lib@v0.0.0-20260913230849-1c87b9904f51` and includes `health_api.go`, and
the image was built 2026-10-24T03:38Z, two minutes after the branch tip
`edde375` (03:36Z). There is no revert commit: `main` simply never received the
work.

Consequence: merging the branch is needed for reproducibility, but **it is not a
fix for the current leak** — the leak happened with the intervention already
deployed.

Note: the branch already carries its own `AGENTS.md` ("Agent notes"). When
merging, reconcile the two files instead of overwriting either.

## Known operational incident: goroutine leak (November 2026)

**Symptom.** Host load average stuck around 3.3 on a 3-vCPU machine; the relay
container consumed ~236% CPU and 1.2 GiB RSS, the host went to 155 MB free with
350 MB in swap, and the service itself reported `status=degraded state=YELLOW`.

**Evidence gathered (2026-11-09, production `relaysv2`).**

- Container uptime 2 days 17 h with 0 restarts.
- Goroutine count grew monotonically: 29,162 → 31,102 → 32,840 → 47,800 in
  24 h, i.e. roughly 750 new goroutines per hour.
- 855 `[WARN] query timed out after 5s` and 716 health warnings in 6 h.
- Container had **no CPU or memory limits** and ran the `:latest` tag.
- Disk I/O was negligible (`read_bytes` ~167 MB), so this is CPU/goroutine
  pressure, not storage.
- The container handled ~98 GB received / 29 GB transmitted since start, so the
  load correlates with query traffic volume.

**Action taken.** The container was restarted. Load fell from 3.05 to 1.77,
container memory from 1.1 GiB to 30 MiB, and free host memory rose from 155 MB
to ~1 GB. This is a temporary relief, not a fix.

**Conclusion.** The growth pattern and the recurring 5-second query timeouts
point to goroutines that are created per query/fanout and never released when
the upstream call times out.

**Important correction.** The earlier intervention for this class of bug
(`4788bcb`, plus the `nostr-lib` bump in `1fbd93b`) is **already in production**
— the deployed image was built from `test/nostr-lib-fanout`. It fixed a
different, earlier leak (go-nostr `dispatchEvent` per mirrored event, slow
`BroadcastEvent`, idle unauthenticated clients) and added a 256-connection cap,
an 8 s AUTH timeout and goroutine dumps. The current growth (29k → 47.8k in
24 h) happened *with* all of that active, so it is a remaining path, not a
regression from a revert. The leak is proportional to query traffic, so the
service degrades after a couple of days and needs a restart until the root
cause is fixed.

## What to investigate next

1. Enable `GOROUTINE_DUMP=on` (the containment added in `4788bcb` already logs a
   dump when the count reaches YELLOW/RED) and capture one dump before the next
   restart. The dump is the fastest way to identify the remaining leak path.
2. The fanout/query path: every timed-out upstream query must have a bounded
   context, a guaranteed `cancel()`, and no channel/goroutine left waiting.
3. `[WARN] fanout` messages in the logs — they appear right before the degraded
   state and are the best behavioural lead.
4. Whether the dropped-event path in the dropping mirror leaves goroutines when
   the ingest queue is full.
5. Reproduce locally with a slow/unresponsive upstream relay and watch
   `goroutines=N` in the heartbeat.

## Rules for agents

- **Do not** raise CPU/memory limits, add restart-on-high-memory, or pin an
  older image as a way to "fix" this incident. Those may be valid operational
  mitigations, but they are decisions for the maintainer and must not replace
  the diagnosis.
- **Do not** change the production compose or the image tag without an explicit
  request; the deployment lives outside this repository.
- Treat `query timed out after 5s` and `fanout` warnings as symptoms worth
  reporting, not noise to silence.
- When proposing a fix, include how to reproduce and how to measure the
  goroutine count before/after.
- Prefer read-only diagnostics first: `docker logs`, `docker stats`,
  `/proc/<pid>/io`, `runtime.NumGoroutine()` from the heartbeat lines.

## Useful read-only commands

```bash
# goroutine trend from the container logs
docker logs --since 24h <container> 2>&1 | grep -o 'goroutines=[0-9]*' | awk -F= '{print $2}' | tail -20

# current resource usage
docker stats --no-stream <container>

# host pressure
cat /proc/loadavg; free -m
```
