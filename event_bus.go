package swarm

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type EventBus struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

func (bus *EventBus) Subscribe(eventType string) <-chan Event {
	ch := make(chan Event, 10)
	bus.mu.Lock()
	bus.subscribers[eventType] = append(bus.subscribers[eventType], ch)
	bus.mu.Unlock()
	return ch
}

func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	slog.Info("", slog.String("Event", event.Type()), slog.String("Details", event.String()))

	subs := bus.subscribers[event.Type()]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (bus *EventBus) Unsubscribe(eventType string, ch <-chan Event) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	subs := bus.subscribers[eventType]
	for i, subscriber := range subs {
		if subscriber == ch {
			close(subscriber)
			bus.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

type Event interface {
	Type() string
	String() string
}

type NodeConnectedEvent struct {
	Id   string
	Addr string
}

func (e NodeConnectedEvent) Type() string {
	return "NodeConnected"
}

func (e NodeConnectedEvent) String() string {
	return fmt.Sprintf("{ Id: %s, Addr: %s }", e.Id, e.Addr)
}

type NodeDisconnectedEvent struct {
	Id   string
	Addr string
}

func (e NodeDisconnectedEvent) Type() string {
	return "NodeDisconnected"
}

func (e NodeDisconnectedEvent) String() string {
	return fmt.Sprintf("{ Id: %s, Addr: %s }", e.Id, e.Addr)
}

type PingLatencyEvent struct {
	Id string
	Addr    string
	Latency time.Duration
}

func (e PingLatencyEvent) Type() string {
	return "PingLatency"
}

func (e PingLatencyEvent) String() string {
	return fmt.Sprintf("{ Id: %s, Addr: %s, Latency: %f }", e.Id, e.Addr, e.Latency.Seconds())
}
