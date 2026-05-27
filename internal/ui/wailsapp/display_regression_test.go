package wailsapp

import (
	"testing"

	"github.com/muhomor/muhomor/internal/ui/model"
)

func TestRegression_SanitizeConnectionForQuit_stripsErrors(t *testing.T) {
	in := ConnectionDTO{
		Connected:    true,
		ErrorText:    "daemon not running",
		ActivityText: "Подключение…",
		Busy:         true,
		Connecting:   true,
		PingError:    true,
	}
	out := sanitizeConnectionForQuit(in)
	if out.ErrorText != "" || out.ActivityText != "" || out.Busy || out.Connecting || out.PingError {
		t.Fatalf("sanitized: %#v", out)
	}
	if out.StatusTitle != "Подключено" {
		t.Fatalf("title=%q", out.StatusTitle)
	}
}

func TestRegression_ConnectionDTO_keepsDistinctActivityWhenConnected(t *testing.T) {
	dto := connectionDTO(model.ConnectionUI{
		Connected:    true,
		ActivityText: "TCP 12/64",
		ProbeText:    "Probe: alive 1 / 10",
	})
	if dto.ActivityText != "TCP 12/64" {
		t.Fatalf("activity=%q", dto.ActivityText)
	}
}
