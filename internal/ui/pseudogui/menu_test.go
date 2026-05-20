package pseudogui

import "testing"

func TestParseChoice(t *testing.T) {
	tests := []struct {
		in   string
		want MenuAction
	}{
		{"1", ActionStatus},
		{"8", ActionUpdateInstall},
		{"q", ActionQuit},
		{"p", ActionProfiles},
		{"H", ActionChain},
	}
	for _, tc := range tests {
		got, ok := ParseChoice(tc.in)
		if !ok || got != tc.want {
			t.Fatalf("%q: got %q ok=%v", tc.in, got, ok)
		}
	}
}
