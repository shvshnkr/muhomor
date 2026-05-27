package selector

import (
	"testing"

	"github.com/muhomor/muhomor/internal/configgen"
	"github.com/muhomor/muhomor/internal/probe/blexit"
)

func TestBlGroupNameForURL(t *testing.T) {
	if blGroupNameForURL("https://api.telegram.org/", 0) != configgen.PretestBLGroupTG {
		t.Fatal("telegram group")
	}
	if blGroupNameForURL("https://web.whatsapp.com/", 1) != configgen.PretestBLGroupWA {
		t.Fatal("whatsapp group")
	}
	if blGroupNameForURL("https://example.com/", 2) != "MUHOMOR_BL_2" {
		t.Fatal("generic group")
	}
	_ = blexit.TestCap
}
