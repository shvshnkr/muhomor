package paths

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Default mihomo external-controller (REST API). Not the mixed proxy port.
const DefaultMihomoControllerHost = "127.0.0.6"

const DefaultControllerPort = 9090
const DefaultDaemonAPIPort = 8751
const DefaultMixedBindHost = "127.0.0.1"

// MihomoControllerHost is where mihomo listens for /proxies, /delay, etc.
// Override: MUHOMOR_MIHOMO_CONTROLLER_HOST (e.g. 127.0.0.6 to avoid clash with other mihomo on :9090).
func MihomoControllerHost() string {
	if v := strings.TrimSpace(os.Getenv("MUHOMOR_MIHOMO_CONTROLLER_HOST")); v != "" {
		return v
	}
	// Legacy alias
	if v := strings.TrimSpace(os.Getenv("MUHOMOR_LOOPBACK_HOST")); v != "" {
		return v
	}
	// WSL2: mihomo may listen on 127.0.0.6 but local HTTP clients time out (mixed :2181 on 127.0.0.1 works).
	if runtime.GOOS == "linux" && IsWSL() {
		return DefaultMixedBindHost
	}
	return DefaultMihomoControllerHost
}

// IsWSL reports Linux-on-WSL (Microsoft kernel in /proc/version).
func IsWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

func MihomoControllerAddr(port int) string {
	return fmt.Sprintf("%s:%d", MihomoControllerHost(), port)
}

// DefaultExternalController is mihomo REST API (backend only, do not publish to LAN).
func DefaultExternalController() string {
	return MihomoControllerAddr(DefaultControllerPort)
}

// MixedBindHost is bind-address for mixed/socks/http (user-facing proxy port).
// Override: MUHOMOR_MIXED_BIND_HOST. Use 0.0.0.0 via settings Allow LAN, not this constant.
func MixedBindHost() string {
	if v := strings.TrimSpace(os.Getenv("MUHOMOR_MIXED_BIND_HOST")); v != "" {
		return v
	}
	return DefaultMixedBindHost
}

// DefaultDaemonTCP is muhomor ctl HTTP when Unix socket is unavailable (local only).
func DefaultDaemonTCP() string {
	return fmt.Sprintf("%s:%d", DefaultMixedBindHost, DefaultDaemonAPIPort)
}
