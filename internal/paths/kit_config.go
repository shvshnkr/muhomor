package paths

import "path/filepath"

// KitConfigDir returns config/ beside a portable kit root.
func KitConfigDir() (string, bool) {
	root, ok := PortableKitRoot()
	if !ok {
		return "", false
	}
	return filepath.Join(root, "config"), true
}

// KitSubscriptionsFile is config/subscriptions.txt in a portable kit.
func KitSubscriptionsFile() (string, bool) {
	dir, ok := KitConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "subscriptions.txt"), true
}
