package web

import "sync"

// EventType identifies the kind of instance lifecycle event.
type EventType string

const (
	EventCreated   EventType = "created"
	EventStarted   EventType = "started"
	EventStopped   EventType = "stopped"
	EventDestroyed EventType = "destroyed"
)

// Event represents an instance lifecycle change.
type Event struct {
	Type EventType `json:"type"`
	Name string    `json:"name"`
}

// EventBus is a simple in-process pub/sub for instance events.
type EventBus struct {
	mu   sync.RWMutex
	subs map[chan Event]struct{}
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{subs: make(map[chan Event]struct{})}
}

// Subscribe returns a channel that receives events.
// Call Unsubscribe when done to avoid leaks.
func (b *EventBus) Subscribe() chan Event {
	ch := make(chan Event, 16)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (b *EventBus) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}

// Publish sends an event to all subscribers (non-blocking).
func (b *EventBus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
			// Subscriber too slow — drop event.
		}
	}
}
