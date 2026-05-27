package mihomo

import "testing"

func TestIsDialTimeoutLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{`level=warning msg="dial PROXY (tcp): i/o timeout"`, true},
		{`level=info msg="Initial configuration complete"`, false},
		{`level=error msg="can't download MMDB"`, false},
		{`dial tcp 1.2.3.4:443: connection refused`, false},
	}
	for _, tc := range cases {
		if got := IsDialTimeoutLine(tc.line); got != tc.want {
			t.Errorf("IsDialTimeoutLine(%q) = %v want %v", tc.line, got, tc.want)
		}
	}
}
