package model

import "strings"

// ProxyDisplayLabel returns the name shown in profile lists, not the mihomo proxy tag.
func ProxyDisplayLabel(profileName, proxyName string) string {
	if strings.TrimSpace(profileName) != "" {
		return profileName
	}
	return proxyName
}
