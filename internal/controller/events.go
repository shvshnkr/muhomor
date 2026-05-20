package controller

import (
	"sync"
	"time"

	"github.com/muhomor/muhomor/internal/api"
)

// EventHub broadcasts daemon events to SSE subscribers.
type EventHub struct {
	mu   sync.RWMutex
	subs map[chan api.Event]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[chan api.Event]struct{})}
}

func (h *EventHub) Subscribe() chan api.Event {
	ch := make(chan api.Event, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan api.Event) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *EventHub) Publish(e api.Event) {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().UnixMilli()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}
