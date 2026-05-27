# Deprecated: Fyne GUI

This package is **deprecated** as of 2026-05-25. Desktop GUI is **Wails** (`internal/ui/wailsapp`, `frontend/`).

- Build legacy Fyne: `go build -tags fyne -tags cgo -ldflags "-H windowsgui" -o muhomor-gui-fyne.exe ./cmd/muhomor-gui`
- Default GUI: `wails build` → `muhomor-gui.exe`

Scheduled for removal after one release with Wails parity.
