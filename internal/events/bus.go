package events

import (
	"sync"
)

const defaultBufferSize = 100

// Bus is a simple pub/sub event bus
type Bus struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
	bufferSize  int
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[chan Event]struct{}),
		bufferSize:  defaultBufferSize,
	}
}

// Subscribe returns a channel that receives events
func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, b.bufferSize)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel
func (b *Bus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find and remove the channel
	for sub := range b.subscribers {
		if sub == ch {
			delete(b.subscribers, sub)
			close(sub)
			return
		}
	}
}

// Publish sends an event to all subscribers
// Non-blocking: if a subscriber's buffer is full, the event is dropped for that subscriber
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, drop event for this subscriber
		}
	}
}

// SubscriberCount returns the number of active subscribers
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Close closes all subscriber channels
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, ch)
	}
}
