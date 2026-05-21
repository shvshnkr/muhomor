# Desktop Multipath Aggregation

Desktop-first adaptive multi-channel selection without mandatory VPS bonding.

## Stack choice

- **Primary:** userspace multipath in muhomor (`internal/aggregate`) on top of selector/probe/fallback.
- **Not used as primary:** OpenMPTCProuter / MLVPN (require VPS terminator for true packet bonding).

## Feature flags (default off)

| KV | Default | Meaning |
|----|---------|---------|
| `multipath_enabled` | `false` | Use channel scoring instead of delay-only ranking |
| `multipath_preset` | `normal` | `low` / `normal` / `high` worker/channel caps |
| `multipath_wl_emergency_only` | `true` | WL builtin only when subscription pool degraded (multipath on) |
| `wl_builtin_connect_enabled` | `false` | H22-retry to free trojan pool when subs dead; **off = subscriptions only** |
| `probe_builtin_fallback_max_pct` | `0` | Builtin share in queue when wl connect on (set 25 if you enable WL) |

Enable via **GUI → Расширенный режим → Настройки → Multipath**, or SQL:

```sql
INSERT OR REPLACE INTO kv (key, value) VALUES ('multipath_enabled', 'true');
INSERT OR REPLACE INTO kv (key, value) VALUES ('multipath_preset', 'normal');
```

## Channel metrics (migration v7)

Per profile: `ch_ewma_goodput_kbps`, `ch_ewma_loss_permille`, `ch_ewma_jitter_ms`, `ch_ewma_queue_delay_ms`, `ch_channel_state`, `ch_last_sample_at`.

States: `healthy`, `congested`, `degraded`, `emergency`.

## Narrow-pipe case

Low RTT + low goodput → higher score penalty (`narrow_pipe` / `congested`), so overloaded servers are not picked ahead of high-goodput channels.

## API

`GET /v1/service/status` → `multipath` object (active/healthy channels, last reason).

## Test matrix

| Case | Expect |
|------|--------|
| WL-only network | WL allowed in pool; emergency policy respected |
| Low RTT, low goodput | Ranked below high-goodput channels |
| Good internet, 3–6 alive channels | Channel pool up to preset max; bulk weights spread |
| Flapping / health fail | Loss EWMA rises; state degrades; fallback |
| 2K + warm/scheduler | No regression; flags off = legacy path |

## Benchmarks

```powershell
go test ./internal/aggregate/ -bench=. -benchmem -count=1
```

## KPI targets

- Multi-stream goodput uplift toward 200 Mbit/s on strong uplink (environment dependent).
- Interactive p95 latency not worse than single-path baseline.
- WL share bounded by `probe_builtin_fallback_max_pct` in normal mode.
