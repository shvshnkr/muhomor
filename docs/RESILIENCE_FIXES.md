# Resilience fixes (probe, subs, handoff)

## Probe / cemetery

- `UpdateProfileProbe` merges with existing meta; URL fail no longer resets cemetery to `unknown`.
- Background due queue: empty `probe_next_probe_at` only for `unknown`; order by `probe_last_checked_at` (fair cemetery rotation).
- Live connect excludes `cemetery` from pool (handoff priority IDs still allowed).
- `warmMarkStale` uses scheduler-aligned states + `NextProbeAfter`.

## Subscriptions

- After successful fetch: `PruneGroupProfilesNotInURIs` removes stale URIs (builtin WL rows kept).
- Connect/scheduler/maintenance: WL vs open refresh split; errors logged (`H29-connect`, scheduler).
- Rule assets: marker written only if at least one YAML downloaded.

## Network / adapt

- `ReachabilityCache.Invalidate` on handoff and stop.
- `probeFresh` updates cache on connect/adapt probe.
- Adapt: debounce waits instead of skip; H22 WL retry; `CancelAll` on stop.
- `Prepare`/`tcpProbeAll`/`ephemeralURLTest` respect cancel + connect/adapt generation.

## Fallback

- `TryMoveFallback`: if current ID not in queue, resume at saved index (not head).
- Builtin cap: excess builtins moved to tail, not dropped from queue.

## Warm select

- Warm list intersected with connect `buildPool`.
- `ListWarmAliveProfiles` requires non-empty `probe_last_ok_at`.
