# Publishing checklist

Use before the first public `git push` or release tag.

## Automated checks

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify-publish.ps1
```

```bash
./scripts/verify-publish.sh
```

## Manual checklist

- [ ] `go test ./...` passes with `CGO_ENABLED=0`
- [ ] No `agent_debug`, `agentDebugLog`, or hard-coded `C:\Users\...` paths in tracked files
- [ ] No `debug-*.log`, `dist/`, `*.db`, `*.db-wal`, or assembled `muhomor-kit*` in the commit
- [ ] `kit/config/subscriptions.txt` has no private subscription URLs (use `subscriptions.example.txt` as template)
- [ ] `LICENSE` (MIT) is present
- [ ] CI workflow `.github/workflows/ci.yml` is green on the PR (local: `scripts/ci-test-matrix.ps1`)

## What stays out of git

| Path | Reason |
|------|--------|
| `dist/`, `muhomor-kit*/` | Built portable kits |
| `bin/mihomo*` | Download via `scripts/fetch-mihomo-*.ps1` |
| `data/`, `*.db` | Runtime state |
| `AI/workflowlog.md` | Local agent journal |
| `.cursor/` | Editor-local rules (optional for contributors) |

## Build notes

- **Daemon / CLI:** no CGO — `go build ./cmd/muhomor`
- **GUI:** CGO + gcc — see [README.md](../README.md) and [AGENTS.md](../AGENTS.md)
- Portable kits: `scripts/pack-windows-kit.ps1`, `scripts/pack-linux-kit.ps1` (output under `dist/`)
