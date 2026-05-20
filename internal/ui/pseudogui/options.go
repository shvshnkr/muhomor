package pseudogui

import "strconv"

// Options for pseudo-GUI launch (Linux / Windows terminal).
type Options struct {
	DaemonArgs []string
}

// DaemonArgsFromCLI builds flags passed to `muhomor --daemon` when auto-starting.
func DaemonArgsFromCLI(serviceMode string, mixedPort int, proxyAuth string, routeQuick int) []string {
	var args []string
	if serviceMode != "" {
		args = append(args, "--service-mode", serviceMode)
	}
	if mixedPort > 0 {
		args = append(args, "--mixed-port", strconv.Itoa(mixedPort))
	}
	if proxyAuth != "" {
		args = append(args, "--proxy-auth", proxyAuth)
	}
	if routeQuick >= 0 {
		args = append(args, "--route-quick-profile", strconv.Itoa(routeQuick))
	}
	return args
}
