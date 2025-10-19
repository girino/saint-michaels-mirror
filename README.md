# Espelho de São Miguel — Relay Agregator

The Espelho de São Miguel is a Nostr relay built on khatru that acts as a mirror and aggregator for relayed events. Its name and brand follow a small myth:

> The Espelho de São Miguel is the sacred mirror that stands between worlds, where every message is received, reflected, and transmitted without distortion under the Archangel’s vigilant gaze. It unites the power of Exu, opener of paths, with the harmony of Ibeji, the divine twins, ensuring that all light crossing its surface returns as truth.

This README explains what the project does, how it models the "mirror" metaphor in technical terms, and how to configure and run the relay.

## Conceptual mapping — myth → implementation
- Mirror: published events are accepted and forwarded (mirrored) to a set of configured remote relays. The relay attempts to faithfully transmit events without modification.
- Archangel (messenger): the Archangel represents the relay's role as a messenger from a higher authority — the relay's job is to receive and relay messages faithfully. The Archangel metaphor emphasizes reliable, accountable transmission and signing of metadata when configured.
- Ibeji (the twins, copying and distribution): the Ibeji twins symbolize mirroring and duplication — the relay's copying behavior (forwarding, merging responses and returning mirrored results). In Afro‑Brazilian syncretism the twins are associated with Cosme e Damião and the practice of giving out candies to children; metaphorically this maps to the relay's role in distributing events and copies to connected clients and remotes — offerings shared with the community.
- Exu (opener of paths and messenger of the orixás): Exu represents the opening of outgoing paths and the relay's active role in speaking to other relays. In the mythic mapping Exu is associated with the messenger function of Saint Michael and the practical network actions: probing remote endpoints and discovering their capabilities (NIP-11).

## Key behaviors (what Espelho does)
- Accepts incoming PUBLISH messages and forwards them (mirrors) to the configured list of `PUBLISH_REMOTES`.
- Answers REQ (query) requests by querying configured `QUERY_REMOTES` and returning results to clients.
- Probes remote relays' root (`/`) with the header `Accept: application/nostr+json` to read NIP-11 metadata and determine if a remote supports NIP-45 (counting). The relay only uses CountMany against remotes that advertise NIP-45.
- Prevents forwarding of khatru internal QueryEvents (except the exact internal filter adding.kind=5/#e) to avoid leaking internal bookkeeping queries.

## Quickstart — configuration
Copy or edit the supplied `.env` or `example.env` as needed. The important variables are:

- `QUERY_REMOTES` — comma-separated list of wss:// or ws:// remotes used to answer REQ queries.
- `PUBLISH_REMOTES` — comma-separated list of remotes used to forward PUBLISH events (if used).
- `ADDR` — address to listen on (default `:3337`).
- `VERBOSE` — set `1` to enable verbose logging.
- `RELAY_SERVICE_URL` — base URL advertised in NIP-11 (optional).
- `RELAY_NAME` — display name; in this project set to `Espelho de São Miguel` by default.
- `RELAY_DESCRIPTION` — a short blurb; the mythic paragraph above is used in `.env` by default.
- `RELAY_CONTACT` — contact `npub` for the relay maintainer.
- `RELAY_SECKEY` — relay secret key (nsec bech32 or hex) used for signing NIP-11 metadata when present.
- `RELAY_ICON` / `RELAY_BANNER` — paths to static assets served under `/static/`.

Example (already included in the repo as `.env`):

```
QUERY_REMOTES=wss://wot.girino.org,wss://nostr.girino.org
ADDR=:3337
VERBOSE=1
RELAY_SERVICE_URL=https://agregator.girino.org
RELAY_NAME="Espelho de São Miguel"
RELAY_DESCRIPTION="The Espelho de São Miguel is the sacred mirror..."
RELAY_CONTACT=npub18... (your contact npub)
RELAY_SECKEY=nsec1... (your secret or hex)
RELAY_ICON=/static/icon.png
RELAY_BANNER=/static/banner.png
```

## Run
Build and run the binary (Go toolchain required):

```bash
go build -o bin/khatru-relay ./cmd/khatru-relay
VERBOSE=1 ./run.sh
```

The relay listens on the address set by `ADDR`. Static assets are served from `cmd/khatru-relay/static/` and the homepage is available at `/`.

## Endpoints and UI
- `/` — homepage describing the relay; shows NIP-11 metadata, contact and badges.
- `/stats` — numeric counters and metrics (separate counters for query vs count operations, forwards succeeded/failed, etc.).
- `/static/*` — static files, including favicons and logos.

## Forwarding and querying details
- Publish forwarding: when a PUBLISH arrives the relay records and then (concurrently) attempts to forward the event to the remotes in `PUBLISH_REMOTES`. Forward attempts and results increment atomic counters (attempts, successes, failures) visible on `/stats`.
- Querying: when clients send REQ, the relay uses a pool to query the configured `QUERY_REMOTES`. Responses are merged and returned to the client. Internal khatru QueryEvents are recognized (exact internal filter adding.kind=5/#e) and are not forwarded to external remotes.
- Counting: the relay will only call CountMany on remotes that advertize NIP-45 in their NIP-11 JSON. This is determined by probing the remote root (`/`) with `Accept: application/nostr+json` at start-up (and periodically, if configured).

## Security and privacy notes
- The relay respects the boundary between internal bookkeeping queries and user queries by short-circuiting internal filters.
- Private keys (`RELAY_SECKEY`) must be kept secure. The service will use the secret key to sign NIP-11 advertised metadata if provided.

## Metrics
The service exposes atomic counters for:
- publishAttempts / publishSuccesses / publishFailures
- queryRequests / queryInternal / queryExternal / queryEventsReturned / queryFailures
- countRequests / countInternal / countExternal / countEventsReturned / countFailures

These counters are surfaced in `/stats` for monitoring and transparency (the "Ibeji" balance: separate counters for query vs count operations).

## Development notes
- Favicon and image assets are generated with a small tool in `tools/icon-resize/` that produces rounded PNGs used under `cmd/khatru-relay/static/favicons/`.
- The relay code uses a `nostr.SimplePool` for querying and a guard that probes remotes for NIP-11 capabilities before using count endpoints.

## Troubleshooting
- If queries return no results, ensure `QUERY_REMOTES` is reachable and supports the filters you send.
- If CountMany looks disabled, the relay likely did not find NIP-45 advertised in a remote's NIP-11; check remote `/` responses.

## Contributing
- Pull requests welcome. Please follow Go project conventions. Tests for relay logic (especially the short-circuiting of internal QueryEvents and NIP-11 gating for CountMany) are highly appreciated.

## License
This repository is released under the license listed in `LICENSE` (if present). If you add proprietary assets, document licensing in `/assets/LICENSE`.

---
_Espelho de São Miguel — the mirror that returns light as truth._
# relay-agregator

Local repository created by assistant.
