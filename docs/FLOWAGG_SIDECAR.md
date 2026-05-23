# Flow Aggregation Sidecar (spec)

MVP uses mihomo `PROXY_BULK` (`type: load-balance`). This document defines the follow-up local sidecar for finer control.

## Goals

- Default path: many-flow throughput across 2–8 VPN legs without external VPS.
- Sticky per-flow mapping; no mid-flow leg migration unless hard failure.
- Control plane stays in muhomor (`channel_pool`, `BulkWeights`, health metrics).

## Architecture

```text
Apps -> mihomo (TUN/mixed) -> flowagg sidecar (127.0.0.1:socks)
                              -> leg1..legN (per-profile outbounds or upstream socks)
```

## Control API (local, authenticated)

| Method | Path | Body |
|--------|------|------|
| PUT | `/v1/pool` | `{ "profile_ids": [26, 30, ...], "primary_id": 26 }` |
| PUT | `/v1/policy` | `{ "mode": "least_loaded", "sticky_ttl_sec": 300 }` |
| PUT | `/v1/weights` | `{ "26": 400, "30": 350, ... }` (permille) |
| POST | `/v1/health` | per-leg fail/latency/goodput samples |
| GET | `/v1/status` | active legs, inflight, fallback reason |

## Scheduler

- **Default:** weighted least-loaded + sticky TTL per 5-tuple / flow key.
- **Never** move an active flow to another leg unless the leg is marked dead.
- Cooldown: exponential 10s → 30s → 60s after hard failures.
- Hysteresis: require N consecutive failures before removing a leg from pool.

## Integration with muhomor

1. On connect, muhomor writes pool + weights from `aggregate.ScheduleResult`.
2. Sidecar exposes single upstream to mihomo config (`PROXY` -> `127.0.0.1:port`).
3. Status API mirrors sidecar `GET /v1/status` into `bulk_*` fields.

## Rollout

1. Ship mihomo `PROXY_BULK` MVP (current).
2. Optional sidecar behind `aggregation_mode=sidecar` (future flag).
3. Benchmark: parallel downloads vs single-file speedtest (expect uplift only on parallel).
