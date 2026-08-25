package notify

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type stubSender struct {
	mu   sync.Mutex
	n    int
	fail bool
}

func (s *stubSender) Send(_ context.Context, _ Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	if s.fail {
		return errors.New("relay refused")
	}
	return nil
}

func (s *stubSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

func TestDispatcherDeliversWhatItIsGiven(t *testing.T) {
	s := &stubSender{}
	d := NewDispatcher(s)

	d.Notify(context.Background(), Welcome("a@b.test", "aino"))
	d.Close()

	if s.count() != 1 {
		t.Errorf("delivered %d, want 1", s.count())
	}
}

func TestASendFailureIsSwallowed(t *testing.T) {
	s := &stubSender{fail: true}
	d := NewDispatcher(s)

	d.Notify(context.Background(), Welcome("a@b.test", "aino"))
	d.Close()

	if s.count() != 1 {
		t.Errorf("attempted %d sends, want 1", s.count())
	}
}

func TestNotifyAfterCloseDoesNotPanic(t *testing.T) {
	d := NewDispatcher(&stubSender{})
	d.Close()

	d.Notify(context.Background(), Welcome("a@b.test", "aino"))
	d.Close()
}

func TestNotifyDuringCloseIsRaceFree(t *testing.T) {
	s := &stubSender{}
	d := NewDispatcher(s)

	var wg sync.WaitGroup
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Notify(context.Background(), Welcome("a@b.test", "aino"))
		}()
	}

	go d.Close()
	wg.Wait()
	d.Close()

	t.Logf("delivered %d of 200 (drops during shutdown are expected)", s.count())
}
