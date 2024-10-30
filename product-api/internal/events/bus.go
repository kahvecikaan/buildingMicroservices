package events

import "sync"

// Event is a generic type placeholder for any event type
type Event any

// Subscriber is a channel that transports events of type T
type Subscriber[T Event] chan T

type EventBus[T Event] struct {
	subscribers map[Subscriber[T]]struct{}
	mutex       sync.RWMutex
}

func NewEventBus[T Event]() *EventBus[T] {
	return &EventBus[T]{
		subscribers: make(map[Subscriber[T]]struct{}),
	}
}

func (bus *EventBus[T]) Subscribe() Subscriber[T] {
	// create a buffered channel of type T with capacity 100
	ch := make(Subscriber[T], 100)
	bus.mutex.Lock()
	bus.subscribers[ch] = struct{}{}
	bus.mutex.Unlock()
	return ch
}

func (bus *EventBus[T]) Unsubscribe(ch Subscriber[T]) {
	bus.mutex.Lock()
	delete(bus.subscribers, ch)
	bus.mutex.Unlock()
	close(ch)
}

// Publish broadcasts an event of type T to all registered subscribers
func (bus *EventBus[T]) Publish(event T) {
	bus.mutex.RLock()
	defer bus.mutex.RUnlock()
	for subscriber := range bus.subscribers {
		select {
		case subscriber <- event:
			// event sent successfully
		default:
			// buffer is full; consider logging or handling?
		}
	}
}
