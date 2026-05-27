package profileclass

import (
	"testing"

	"github.com/muhomor/muhomor/internal/store"
)

func TestRuExitMarkedByName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Russia [05]", true},
		{"🇷🇺 Fast", true},
		{"FR Paris", false},
		{"[BL] DE", false},
	}
	for _, tc := range cases {
		if got := RuExitMarkedByName(tc.name); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsRuExitMarked_dbFlag(t *testing.T) {
	p := store.Profile{Name: "Paris", RuExitMarked: true}
	if !IsRuExitMarked(p) {
		t.Fatal("expected db flag")
	}
}

func TestIsBLModeUplink(t *testing.T) {
	if !IsBLModeUplink(store.Profile{Name: "Anycast [BL]"}) {
		t.Fatal("expected [bl] tag")
	}
	if !IsBLModeUplink(store.Profile{Name: "Russia [01]", RuExitMarked: true}) {
		t.Fatal("expected ru_exit")
	}
	if IsBLModeUplink(store.Profile{Name: "DE node"}) {
		t.Fatal("plain DE should not be uplink")
	}
}

func TestIsBLSubscriptionMarked(t *testing.T) {
	p := store.Profile{Name: "DE [BL] #1"}
	if !IsBLSubscriptionMarked(p) {
		t.Fatal("expected [bl] tag")
	}
}
