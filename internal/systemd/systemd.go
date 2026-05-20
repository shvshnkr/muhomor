package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Run manages systemd user unit (Linux only).
func Run(command, scope string, launcher []string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("systemd: only supported on Linux")
	}
	cmd := strings.TrimSpace(strings.ToLower(command))
	unit := "muhomor.service"
	switch cmd {
	case "install":
		return installUnit(scope, unit, launcher)
	case "uninstall":
		return uninstallUnit(scope, unit)
	case "enable":
		return systemctl(scope, "enable", unit)
	case "disable":
		return systemctl(scope, "disable", unit)
	case "start", "stop", "restart", "status":
		return systemctl(scope, cmd, unit)
	default:
		return fmt.Errorf("unknown systemd command: %s", command)
	}
}

func installUnit(scope, unit string, launcher []string) error {
	dir, err := unitDir(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	execStart := strings.Join(launcher, " ")
	if !strings.Contains(execStart, "--daemon") {
		execStart += " --daemon"
	}
	content := fmt.Sprintf(`[Unit]
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
	path := filepath.Join(dir, unit)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Println("installed", path)
	return systemctl(scope, "daemon-reload")
}

func uninstallUnit(scope, unit string) error {
	dir, err := unitDir(scope)
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dir, unit))
	return systemctl(scope, "daemon-reload")
}

func unitDir(scope string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if scope == "system" {
		return "/etc/systemd/system", nil
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

func systemctl(scope string, args ...string) error {
	cmdArgs := []string{"systemctl"}
	if scope != "system" {
		cmdArgs = append(cmdArgs, "--user")
	}
	cmdArgs = append(cmdArgs, args...)
	c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
