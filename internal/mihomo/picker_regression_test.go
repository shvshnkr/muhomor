package mihomo

import (
	"context"
	"sync"
	"testing"

	"github.com/muhomor/muhomor/internal/paths"
)

func TestRegression_PickerEnsure_concurrentWhileRunning(t *testing.T) {
	dir := t.TempDir()
	p := NewPicker(paths.Layout{RuntimeDir: dir}, "")
	p.mu.Lock()
	p.running = true
	p.client = NewClient(ClientOptions{Controller: "127.0.0.1:8760"})
	p.mu.Unlock()

	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Ensure(context.Background()); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Ensure: %v", err)
	}
	if !p.Available() {
		t.Fatal("picker should stay available after concurrent Ensure")
	}
}
