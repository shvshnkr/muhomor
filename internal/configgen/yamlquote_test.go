package configgen

import (
	"strings"
	"testing"
)

func TestYAMLQuote_brackets(t *testing.T) {
	name := "Anycast_|_[BL]"
	q := yamlQuote(name)
	if !strings.Contains(q, "[BL]") {
		t.Fatalf("quote=%q", q)
	}
	var b strings.Builder
	yamlNameLine(&b, name)
	line := b.String()
	if !strings.Contains(line, `"`) {
		t.Fatalf("expected quoted name line, got %q", line)
	}
}
