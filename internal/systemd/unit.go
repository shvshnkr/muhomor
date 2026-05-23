package systemd

import (
	"fmt"
	"strings"
)

// UnitDir returns the directory for muhomor.service (user or system scope).
func UnitDir(scope string) (string, error) {
	return unitDir(scope)
}

// RenderUnit returns a systemd unit file body for ExecStart launcher.
func RenderUnit(execStart string) string {
	if execStart == "" {
		execStart = "muhomor -dir /var/lib/muhomor --daemon"
	}
	if !containsDaemonFlag(execStart) {
		execStart += " --daemon"
	}
	return fmt.Sprintf(`[Unit]
Description=muhomor VPN daemon
After=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, execStart)
}

func containsDaemonFlag(s string) bool {
	return strings.Contains(s, "--daemon")
}
