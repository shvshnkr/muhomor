package mihomo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/muhomor/muhomor/internal/paths"
)

func TestPickerEnsure_skipsWhenRunning(t *testing.T) {
	dir := t.TempDir()
	p := NewPicker(paths.Layout{RuntimeDir: dir}, "")
	p.mu.Lock()
	p.running = true
	p.client = NewClient(ClientOptions{Controller: "127.0.0.1:8760"})
	p.mu.Unlock()
	if err := p.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure on running picker: %v", err)
	}
}

func TestPickerEnsure_writesBootstrapBeforeStart(t *testing.T) {
	dir, err := os.MkdirTemp("", "picker-bootstrap-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	layout := paths.Layout{RuntimeDir: dir}
	p := NewPicker(layout, filepath.Join(dir, "missing-mihomo"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Ensure(ctx); err == nil {
		// Another picker may already listen on :8760 (kit/dev); reuse is valid.
		p.Stop()
		return
	}
	cfgPath := filepath.Join(dir, "picker", "config.yaml")
	data, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("bootstrap should be written before start failure: %v", readErr)
	}
	if !strings.Contains(string(data), "mixed-port: 2182") {
		t.Fatalf("bootstrap yaml: %q", data)
	}
}
