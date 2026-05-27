# Security

## Do not commit secrets

- VPN subscription URLs that embed credentials or personal tokens
- Imported profile URIs (`vless://`, `trojan://`, etc.) with keys
- Runtime directories: `data/`, `dist/`, assembled kit trees, SQLite `*.db`
- Local debug logs (`debug-*.log`, `daemon-debug*.log`)

Use `kit/config/subscriptions.example.txt` as a template; keep your real links in local `subscriptions.txt` only.

## Reporting issues

Open a GitHub issue for vulnerabilities you are comfortable disclosing publicly. For sensitive reports, describe impact without pasting live credentials or full subscription payloads.
