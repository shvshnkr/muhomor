package model

import "testing"

func TestParseServerDisplay(t *testing.T) {
	cases := []struct {
		in              string
		flag, country   string
		server          string
		showCountry     bool
	}{
		{"🇩🇪 DE Frankfurt #1", "🇩🇪", "Германия", "Frankfurt #1", true},
		{"Germany · Frankfurt", "🇩🇪", "Германия", "Frankfurt", true},
		{"US-NY-01", "🇺🇸", "США", "US-NY-01", true},
		{"Node-42", "🌐", "", "Node-42", false},
	}
	for _, tc := range cases {
		d := ParseServerDisplay(tc.in)
		if d.Flag != tc.flag || d.Country != tc.country || d.ServerName != tc.server || d.ShowCountry != tc.showCountry {
			t.Errorf("%q: got flag=%q country=%q server=%q show=%v; want flag=%q country=%q server=%q show=%v",
				tc.in, d.Flag, d.Country, d.ServerName, d.ShowCountry,
				tc.flag, tc.country, tc.server, tc.showCountry)
		}
	}
}

func TestFormatBps(t *testing.T) {
	if FormatBps(0) != "0 B/s" {
		t.Fatal(FormatBps(0))
	}
	if FormatBps(1300000) != "1.2 MB/s" && FormatBps(1300000) != "1.3 MB/s" {
		// allow rounding variance
		got := FormatBps(1300000)
		if got != "1.2 MB/s" && got != "1.3 MB/s" {
			t.Fatalf("got %s", got)
		}
	}
}
