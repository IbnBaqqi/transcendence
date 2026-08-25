package notify

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	queueSize    = 256
	drainTimeout = 10 * time.Second
)

type Dispatcher struct {
	sender Sender
	queue  chan Message
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

func NewDispatcher(sender Sender) *Dispatcher {
	d := &Dispatcher{
		sender: sender,
		queue:  make(chan Message, queueSize),
	}

	d.wg.Add(1)
	go d.work()

	return d
}

func (d *Dispatcher) Notify(_ context.Context, m Message) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		slog.Warn("notification dropped, dispatcher is shutting down", "kind", m.Kind)
		return
	}

	select {
	case d.queue <- m:
	default:
		slog.Warn("notification dropped, queue is full", "kind", m.Kind, "queue_size", queueSize)
	}
}

func (d *Dispatcher) work() {
	defer d.wg.Done()

	for m := range d.queue {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		err := d.sender.Send(ctx, m)
		cancel()

		if err != nil {
			slog.Error("sending a notification failed",
				"kind", m.Kind, "to", m.To, "error", err)
		}
	}
}

func (d *Dispatcher) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	close(d.queue)
	d.mu.Unlock()

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(drainTimeout):
		slog.Warn("gave up draining notifications", "timeout", drainTimeout)
	}
}
