package server

import (
	"sync"
	"time"
)

// Clock is the app's source of time: the current instant and periodic ticks.
type Clock interface {
	Now() time.Time
	// Every calls fn each period d, never overlapping, until the returned stop func is called; stop waits for a running call.
	Every(d time.Duration, fn func()) (stop func())
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Every(d time.Duration, fn func()) func() {
	ticker := time.NewTicker(d)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fn()
			}
		}
	})
	return func() {
		ticker.Stop()
		close(done)
		wg.Wait()
	}
}
