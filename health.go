// This package provides a series of primitive methods to
// assist you with graceful shutdown implementation.

package health

import (
	"sync"
	"sync/atomic"
)

type Status = int32

const (
	StatusInit Status = iota
	StatusUp
	StatusDown
)

var (
	status  int32
	opState interface{}
	mtx     sync.Mutex
)

// SetUp sets health status to StatusUp and starts health checks.
func SetUp(newOpState interface{}) {
	if !atomic.CompareAndSwapInt32(&status, StatusInit, StatusUp) {
		return
	}

	opState = newOpState
	startChecks()
}

// GetStatus returns health status.
func GetStatus() Status {
	return atomic.LoadInt32(&status)
}

// GetOpState returns application's operational state.
// Operational state is only meant to exist on StatusUp.
func GetOpState() interface{} {
	return opState
}
