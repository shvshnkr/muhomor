package configgen

import (
	"fmt"
	"strconv"
	"strings"
)

// yamlQuote returns a YAML-safe double-quoted string (proxy names may contain [ ], |, emoji).
func yamlQuote(s string) string {
	return strconv.Quote(s)
}

func yamlNameLine(b *strings.Builder, name string) {
	fmt.Fprintf(b, "  - name: %s\n", yamlQuote(name))
}

func yamlProxyRef(b *strings.Builder, name string) {
	fmt.Fprintf(b, "      - %s\n", yamlQuote(name))
}
