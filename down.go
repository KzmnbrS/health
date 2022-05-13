package health

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	downDelay int64
	downFns   []func()

	sigint chan os.Signal
	downWg sync.WaitGroup
)

func init() {
	sigint = make(chan os.Signal)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
}

// SetDown emulates syscall.SIGINT.
func SetDown() {
	if GetStatus() == StatusDown {
		return
	}

	signal.Stop(sigint)
	sigint <- syscall.SIGINT
}

// SetDownDelay sets a delay between a health check failure and down
// functions execution start. This might be useful to give your load
// balancer of choice some time to react.
func SetDownDelay(v time.Duration) {
	atomic.StoreInt64(&downDelay, int64(v))
}

// AddDownFn adds a function to run after an interrupt signal.
func AddDownFn(fn func()) {
	if GetStatus() == StatusDown {
		return
	}

	downWg.Add(1)

	mtx.Lock()
	downFns = append(downFns, fn)
	mtx.Unlock()
}

// WaitDown blocks until either down function list is done or the time ran out.
func WaitDown(timeout time.Duration) {
	<-sigint
	atomic.StoreInt32(&status, StatusDown)

	if atomic.LoadInt64(&downDelay) > 0 {
		time.Sleep(time.Duration(downDelay))
	}

	go func() {
		for _, fn := range downFns {
			fn()
			downWg.Done()
		}
	}()

	downFnsDone := make(chan struct{})
	go func() {
		downWg.Wait()
		downFnsDone <- struct{}{}
	}()

	var timeoutC <-chan time.Time
	if timeout > 0 {
		timeoutC = time.After(timeout)
	}

	select {
	case <-downFnsDone:
	case <-timeoutC:
	}
}
