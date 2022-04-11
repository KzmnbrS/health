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
	isDown    int32
	downDelay int64
	downFns   []func()
	syncMap   []bool
	mtx       sync.Mutex

	sigint chan os.Signal
	wg     sync.WaitGroup
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

		for i, fn := range downFns {
			if syncMap[i] {
				wg.Wait()
			}

			wg.Add(1)
			switch syncMap[i] {
			case true:
				fn()
				wg.Done()

			case false:
				go func() {
					fn()
					wg.Done()
				}()
			}
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

// SetDownDelay sets a delay between a health check failure and down
// functions execution start. This might be useful to give your load
// balancer some time to react.
func SetDownDelay(v time.Duration) {
	atomic.StoreInt64(&downDelay, int64(v))
}

// AddDownFn adds a sync func to run after an interrupt signal.
func AddDownFn(fn func()) {
	addDownFn(fn, true)
}

// AddAsyncDownFn adds an async func to run after an interrupt signal.
func AddAsyncDownFn(fn func()) {
	addDownFn(fn, false)
}

func addDownFn(fn func(), isSync bool) {
	if !Check() {
		return
	}

	mtx.Lock()
	downFns = append(downFns, fn)
	syncMap = append(syncMap, isSync)
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
