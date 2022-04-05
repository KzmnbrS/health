// This package provides a series of primitive methods to
// assist you with graceful health implementation.

package health

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

var (
	downFns []func()
	isDown  int32
	mtx     sync.Mutex

	sigint chan os.Signal
	wg     sync.WaitGroup
)

func init() {
	sigint = make(chan os.Signal)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigint
		atomic.StoreInt32(&isDown, 1)

		wg.Add(len(downFns))
		for _, fn := range downFns {
			fn()
			wg.Done()
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

// AddDownFn adds a function to run after an interrupt signal.
func AddDownFn(fn func()) {
	if !Check() {
		return
	}

	mtx.Lock()
	downFns = append(downFns, fn)
	mtx.Unlock()
}

// WaitDown blocks until either down function list or the context is done.
func WaitDown(ctx context.Context) {
	downFnsDone := make(chan struct{})
	go func() {
		wg.Wait()
		downFnsDone <- struct{}{}
	}()

	select {
	case <-downFnsDone:
	case <-ctx.Done():
	}
}
