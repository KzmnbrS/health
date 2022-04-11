// This package provides a series of primitive methods to
// assist you with graceful shutdown implementation.

package health

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	isDown     int32
	downDelay  int64
	downGroups [][]func()
	mtx        sync.Mutex

	sigint  chan os.Signal
	groupWg sync.WaitGroup
)

func init() {
	sigint = make(chan os.Signal)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigint
		atomic.StoreInt32(&isDown, 1)

		if atomic.LoadInt64(&downDelay) > 0 {
			time.Sleep(time.Duration(downDelay))
		}

		for _, group := range downGroups {
			wg := sync.WaitGroup{}
			wg.Add(len(group))
			for _, fn := range group {
				go func(fn func()) {
					fn()
					wg.Done()
				}(fn)
			}
			wg.Wait()
		}
	}()
}

// Stop emulates syscall.SIGINT.
func Stop() {
	signal.Stop(sigint)
	sigint <- syscall.SIGINT
}

// Check returns true if the process wasn't interrupted by a signal.
func Check() bool {
	return atomic.LoadInt32(&isDown) == 0
}

// SetDownDelay sets a delay between a health check failure and down functions start.
// This is useful to give your load balancer of choice some time to react.
func SetDownDelay(v time.Duration) {
	atomic.StoreInt64(&downDelay, int64(v))
}

// AddDownFn adds a function to run after an interrupt signal plus down delay.
// isSync will join all down functions running before executing the given one.
func AddDownFn(fn func(), isSync bool) {
	if !Check() {
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	if isSync || len(downGroups) == 0 {
		groupWg.Add(1)
		downGroups = append(downGroups, []func(){fn})
		return
	}

	n := len(downGroups)
	downGroups[n-1] = append(downGroups[n-1], fn)
}

// WaitDown blocks until either down function list or the context is done.
func WaitDown(ctx context.Context) {
	downFnsDone := make(chan struct{})
	go func() {
		groupWg.Wait()
		downFnsDone <- struct{}{}
	}()

	select {
	case <-downFnsDone:
	case <-ctx.Done():
	}
}
