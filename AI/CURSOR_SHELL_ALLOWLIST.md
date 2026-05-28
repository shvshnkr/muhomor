# Allowlist для агента: muhomor + kit + mihomo

Скопируй в **Cursor Settings → Agents / Terminal / Command allowlist** (или подтверждай `Run with full permissions` / **all** для этих паттернов).

Без `all` агент **не видит** `dist/muhomor-kit/` (`.cursorignore`) и не может читать логи/останавливать процессы.

---

## 1. Чтение логов и состояния kit (обязательно `all`)

```powershell
# Корень kit (Windows portable)
$repoRoot = (git rev-parse --show-toplevel)
$kit = Join-Path $repoRoot "dist\muhomor-kit"
$cache = "$kit\data\cache"

Get-Content "$cache\daemon-debug.err.log" -Tail 50 -Encoding UTF8
Get-Content "$cache\activity.log" -Tail 30 -Encoding UTF8
Get-Content "$cache\desktop-control-status.txt" -Encoding UTF8
Get-Content "$kit\data\run\mihomo\mihomo-subprocess.log" -Tail 30 -Encoding UTF8

Get-Process muhomor*, mihomo* -ErrorAction SilentlyContinue
netstat -ano | findstr ":8751 :2181 :9090"
```

```powershell
# API демона + mihomo (localhost only)
Invoke-RestMethod "http://127.0.0.1:8751/v1/service/status" -TimeoutSec 10
$sec = ((Get-Content "$kit\data\run\config.yaml" -Encoding UTF8 | Where-Object { $_ -match '^\s*secret:' }) -replace '.*secret:\s*"?([^"]+)"?.*','$1').Trim()
curl.exe -s -m 3 -H "Authorization: Bearer $sec" "http://127.0.0.6:9090/version"
curl.exe -s -m 2 -H "Authorization: Bearer $sec" "http://127.0.0.6:9090/traffic"
curl.exe -s -m 3 -H "Authorization: Bearer $sec" "http://127.0.0.6:9090/proxies/PROXY"
```

**Не использовать для mihomo `/traffic`:** `Invoke-WebRequest` — зависает на стриме (минуты).

---

## 2. Остановка / перезапуск kit

```powershell
cd (git rev-parse --show-toplevel)
powershell -ExecutionPolicy Bypass -File .\scripts\stop-muhomor.ps1
```

```powershell
# Пересборка в dist\muhomor-kit (после правок Go)
go build -o .\dist\muhomor-kit\muhomor.exe .\cmd\muhomor
powershell -ExecutionPolicy Bypass -File .\scripts\build-gui-wails.ps1 -OutFile "dist\muhomor-kit\muhomor-gui.exe" -NoPackKit
```

```powershell
# Полный pack (чистый data/)
powershell -ExecutionPolicy Bypass -File .\scripts\pack-windows-kit.ps1
```

---

## 3. Сборка и тесты (репозиторий)

```powershell
cd (git rev-parse --show-toplevel)
go test ./internal/... -count=1 -timeout 120s
go build -o muhomor.exe .\cmd\muhomor
```

```powershell
cd (Join-Path (git rev-parse --show-toplevel) frontend)
npm install
npm run build
```

```powershell
cd (git rev-parse --show-toplevel)
wails build -platform windows/amd64 -o muhomor-gui.exe
```

**GUI:** только `wails build` / `build-gui-wails.ps1`, не `go build ./cmd/muhomor-gui`.

---

## 4. Сеть (localhost)

Разрешить исходящие на loopback:

| Порт | Сервис |
|------|--------|
| `127.0.0.1:8751` | muhomor daemon REST + SSE |
| `127.0.0.1:2181` | mihomo mixed proxy |
| `127.0.0.6:9090` | mihomo external-controller API |

Права: **`full_network`** или **`all`** — для `curl`, `Invoke-RestMethod`, `go test` с httptest.

---

## 5. `.cursorignore` (опционально)

Чтобы агент читал логи без `all` каждый раз, добавь в конец `.cursorignore`:

```gitignore
!dist/muhomor-kit/data/cache/
!dist/muhomor-kit/data/cache/*.log
!dist/muhomor-kit/data/cache/*.txt
!dist/muhomor-kit/data/run/mihomo/mihomo-subprocess.log
```

`*.db` и бинарники kit по-прежнему игнорируются.

---

## 6. Краткий чеклист «нет коннекта»

1. `desktop-control-status.txt` → `connected=true`?
2. `Invoke-RestMethod …/v1/service/status` → `state=Connected`?
3. `netstat` → один `ESTABLISHED` GUI→`:8751` (не 4+)?
4. mihomo `:2181` LISTENING?
5. В GUI: если API Connected, а пилюля «Ошибка» — устаревший `errorText` (фикс в `display.go` / `Simple.tsx`).
